package server

import (
	"context"

	runtimev1 "github.com/rilldata/rill/proto/gen/rill/runtime/v1"
	"github.com/rilldata/rill/runtime/queries"
	"github.com/rilldata/rill/runtime/server/auth"
)

const _tableHeadDefaultLimit = 25

// Table level profiling APIs.
func (s *Server) TableCardinality(ctx context.Context, req *runtimev1.TableCardinalityRequest) (*runtimev1.TableCardinalityResponse, error) {
	if !auth.GetClaims(ctx).CanInstance(req.InstanceId, auth.ReadProfiling) {
		return nil, ErrForbidden
	}

	q := &queries.TableCardinality{
		TableName: req.TableName,
	}
	err := s.runtime.Query(ctx, req.InstanceId, q, int(req.Priority))
	if err != nil {
		return nil, err
	}
	return &runtimev1.TableCardinalityResponse{
		Cardinality: q.Result,
	}, nil
}

type ColumnInfo struct {
	Name    string
	Type    string
	Unknown int
}

func (s *Server) TableColumns(ctx context.Context, req *runtimev1.TableColumnsRequest) (*runtimev1.TableColumnsResponse, error) {
	if !auth.GetClaims(ctx).CanInstance(req.InstanceId, auth.ReadProfiling) {
		return nil, ErrForbidden
	}

	q := &queries.TableColumns{
		TableName: req.TableName,
	}

	err := s.runtime.Query(ctx, req.InstanceId, q, int(req.Priority))
	if err != nil {
		return nil, err
	}

	return &runtimev1.TableColumnsResponse{
		ProfileColumns: q.Result,
	}, nil
}

func (s *Server) TableRows(ctx context.Context, req *runtimev1.TableRowsRequest) (*runtimev1.TableRowsResponse, error) {
	if !auth.GetClaims(ctx).CanInstance(req.InstanceId, auth.ReadProfiling) {
		return nil, ErrForbidden
	}

	limit := int(req.Limit)
	if limit == 0 {
		limit = _tableHeadDefaultLimit
	}

	q := &queries.TableHead{
		TableName: req.TableName,
		Limit:     limit,
	}

	err := s.runtime.Query(ctx, req.InstanceId, q, int(req.Priority))
	if err != nil {
		return nil, err
	}

	return &runtimev1.TableRowsResponse{
		Data: q.Result,
	}, nil
}
