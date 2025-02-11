package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	gateway "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime"
	"github.com/rilldata/rill/runtime/pkg/graceful"
	"github.com/rilldata/rill/runtime/pkg/observability"
	"github.com/rilldata/rill/runtime/server/auth"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var ErrForbidden = status.Error(codes.Unauthenticated, "action not allowed")

type Options struct {
	HTTPPort        int
	GRPCPort        int
	AllowedOrigins  []string
	ServePrometheus bool
	AuthEnable      bool
	AuthIssuerURL   string
	AuthAudienceURL string
}

type Server struct {
	runtimev1.UnsafeRuntimeServiceServer
	runtimev1.UnsafeQueryServiceServer
	runtime *runtime.Runtime
	opts    *Options
	logger  *zap.Logger
	aud     *auth.Audience
}

var (
	_ runtimev1.RuntimeServiceServer = (*Server)(nil)
	_ runtimev1.QueryServiceServer   = (*Server)(nil)
)

func NewServer(opts *Options, rt *runtime.Runtime, logger *zap.Logger) (*Server, error) {
	srv := &Server{
		opts:    opts,
		runtime: rt,
		logger:  logger,
	}

	if opts.AuthEnable {
		aud, err := auth.OpenAudience(logger, opts.AuthIssuerURL, opts.AuthAudienceURL)
		if err != nil {
			return nil, err
		}
		srv.aud = aud
	}

	return srv, nil
}

// Close should be called when the server is done
func (s *Server) Close() error {
	// TODO: This should probably trigger a server shutdown

	if s.aud != nil {
		s.aud.Close()
	}

	return nil
}

// Ping implements RuntimeService
func (s *Server) Ping(ctx context.Context, req *runtimev1.PingRequest) (*runtimev1.PingResponse, error) {
	resp := &runtimev1.PingResponse{
		Version: "", // TODO: Return version
		Time:    timestamppb.New(time.Now()),
	}
	return resp, nil
}

// ServeGRPC Starts the gRPC server.
func (s *Server) ServeGRPC(ctx context.Context) error {
	server := grpc.NewServer(
		grpc.ChainStreamInterceptor(
			observability.TracingStreamServerInterceptor(),
			observability.LoggingStreamServerInterceptor(s.logger),
			observability.RecovererStreamServerInterceptor(),
			grpc_validator.StreamServerInterceptor(),
			auth.StreamServerInterceptor(s.aud),
		),
		grpc.ChainUnaryInterceptor(
			observability.TracingUnaryServerInterceptor(),
			observability.LoggingUnaryServerInterceptor(s.logger),
			observability.RecovererUnaryServerInterceptor(),
			grpc_validator.UnaryServerInterceptor(),
			auth.UnaryServerInterceptor(s.aud),
		),
	)

	runtimev1.RegisterRuntimeServiceServer(server, s)
	runtimev1.RegisterQueryServiceServer(server, s)
	s.logger.Sugar().Infof("serving runtime gRPC on port:%v", s.opts.GRPCPort)
	return graceful.ServeGRPC(ctx, server, s.opts.GRPCPort)
}

// Starts the HTTP server.
func (s *Server) ServeHTTP(ctx context.Context, registerAdditionalHandlers func(mux *http.ServeMux)) error {
	handler, err := s.HTTPHandler(ctx, registerAdditionalHandlers)
	if err != nil {
		return err
	}

	server := &http.Server{Handler: handler}
	s.logger.Sugar().Infof("serving HTTP on port:%v", s.opts.HTTPPort)
	return graceful.ServeHTTP(ctx, server, s.opts.HTTPPort)
}

// HTTPHandler HTTP handler serving REST gateway.
func (s *Server) HTTPHandler(ctx context.Context, registerAdditionalHandlers func(mux *http.ServeMux)) (http.Handler, error) {
	// Create REST gateway
	gwMux := gateway.NewServeMux(gateway.WithErrorHandler(HTTPErrorHandler))
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	grpcAddress := fmt.Sprintf(":%d", s.opts.GRPCPort)
	err := runtimev1.RegisterRuntimeServiceHandlerFromEndpoint(ctx, gwMux, grpcAddress, opts)
	if err != nil {
		return nil, err
	}
	err = runtimev1.RegisterQueryServiceHandlerFromEndpoint(ctx, gwMux, grpcAddress, opts)
	if err != nil {
		return nil, err
	}

	// One-off REST-only path for multipart file upload
	// NOTE: It's local only and we should deprecate it in favor of a cloud-friendly alternative.
	err = gwMux.HandlePath("POST", "/v1/instances/{instance_id}/files/upload/-/{path=**}", auth.GatewayMiddleware(s.aud, s.UploadMultipartFile))
	if err != nil {
		panic(err)
	}

	// One-off REST-only path for file export
	// NOTE: It's local only and we should deprecate it in favor of a cloud-friendly alternative.
	err = gwMux.HandlePath("GET", "/v1/instances/{instance_id}/table/{table_name}/export/{format}", auth.GatewayMiddleware(s.aud, s.ExportTable))
	if err != nil {
		panic(err)
	}

	// Call callback to register additional paths
	// NOTE: This is so ugly, but not worth refactoring it properly right now.
	httpMux := http.NewServeMux()
	if registerAdditionalHandlers != nil {
		registerAdditionalHandlers(httpMux)
	}

	// Add httpMux on gRPC-gateway
	httpMux.Handle("/v1/", gwMux)

	// Add Prometheus
	if s.opts.ServePrometheus {
		httpMux.Handle("/metrics", promhttp.Handler())
	}

	// Build CORS options for runtime server

	// If the AllowedOrigins contains a "*" we want to return the requester's origin instead of "*" in the "Access-Control-Allow-Origin" header.
	// This is useful in development. In production, we set AllowedOrigins to non-wildcard values, so this does not have security implications.
	// Details: https://github.com/rs/cors#allow--with-credentials-security-protection
	var allowedOriginFunc func(string) bool
	allowedOrigins := s.opts.AllowedOrigins
	for _, origin := range s.opts.AllowedOrigins {
		if origin == "*" {
			allowedOriginFunc = func(origin string) bool { return true }
			allowedOrigins = nil
			break
		}
	}

	corsOpts := cors.Options{
		AllowedOrigins:  allowedOrigins,
		AllowOriginFunc: allowedOriginFunc,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		// Set max age to 1 hour (default if not set is 5 seconds)
		MaxAge: 60 * 60,
	}

	// Wrap mux with CORS middleware
	handler := cors.New(corsOpts).Handler(httpMux)

	return handler, nil
}

// HTTPErrorHandler wraps gateway.DefaultHTTPErrorHandler to map gRPC unknown errors (i.e. errors without an explicit
// code) to HTTP status code 400 instead of 500.
func HTTPErrorHandler(ctx context.Context, mux *gateway.ServeMux, marshaler gateway.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	s := status.Convert(err)
	if s.Code() == codes.Unknown {
		err = &gateway.HTTPStatusError{HTTPStatus: http.StatusBadRequest, Err: err}
	}
	gateway.DefaultHTTPErrorHandler(ctx, mux, marshaler, w, r, err)
}
