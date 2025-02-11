package server

import (
	"context"
	"fmt"

	"github.com/rilldata/rill/admin/database"
	"github.com/rilldata/rill/admin/server/auth"
	adminv1 "github.com/rilldata/rill/proto/gen/rill/admin/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (s *Server) GetCurrentUser(ctx context.Context, req *adminv1.GetCurrentUserRequest) (*adminv1.GetCurrentUserResponse, error) {
	// Return an empty result if not authenticated.
	claims := auth.GetClaims(ctx)
	if claims.OwnerType() == auth.OwnerTypeAnon {
		return &adminv1.GetCurrentUserResponse{}, nil
	}

	// Error if authenticated as anything other than a user
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, fmt.Errorf("not authenticated as a user")
	}

	// Owner is a user
	u, err := s.admin.DB.FindUser(ctx, claims.OwnerID())
	if err != nil {
		return nil, err
	}

	return &adminv1.GetCurrentUserResponse{
		User: userToPB(u),
	}, nil
}

// RevokeCurrentAuthToken revokes the current auth token
func (s *Server) RevokeCurrentAuthToken(ctx context.Context, req *adminv1.RevokeCurrentAuthTokenRequest) (*adminv1.RevokeCurrentAuthTokenResponse, error) {
	claims := auth.GetClaims(ctx)
	if claims == nil {
		return nil, status.Error(codes.Unauthenticated, "not authenticated")
	}

	// Error if authenticated as anything other than a user
	if claims.OwnerType() != auth.OwnerTypeUser {
		return nil, fmt.Errorf("not authenticated as a user")
	}
	tokenID := claims.AuthTokenID()

	err := s.admin.DB.DeleteUserAuthToken(ctx, tokenID)
	if err != nil {
		return nil, err
	}

	return &adminv1.RevokeCurrentAuthTokenResponse{
		TokenId: tokenID,
	}, nil
}

func userToPB(u *database.User) *adminv1.User {
	return &adminv1.User{
		Id:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		PhotoUrl:    u.PhotoURL,
		CreatedOn:   timestamppb.New(u.CreatedOn),
		UpdatedOn:   timestamppb.New(u.UpdatedOn),
	}
}

func memberToPB(m *database.Member) *adminv1.Member {
	return &adminv1.Member{
		UserId:    m.ID,
		UserEmail: m.Email,
		UserName:  m.DisplayName,
		RoleName:  m.RoleName,
		CreatedOn: timestamppb.New(m.CreatedOn),
		UpdatedOn: timestamppb.New(m.UpdatedOn),
	}
}

func inviteToPB(i *database.Invite) *adminv1.UserInvite {
	return &adminv1.UserInvite{
		Email:     i.Email,
		Role:      i.Role,
		InvitedBy: i.InvitedBy,
	}
}
