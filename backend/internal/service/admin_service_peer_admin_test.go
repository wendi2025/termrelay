//go:build unit

package service

import (
	"context"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestAdminService_UpdateUser_RejectsPeerAdminProtectedFields(t *testing.T) {
	tests := []struct {
		name  string
		input UpdateUserInput
	}{
		{name: "password", input: UpdateUserInput{Password: "new-secure-password"}},
		{name: "email", input: UpdateUserInput{Email: "other-admin@example.com"}},
		{name: "role", input: UpdateUserInput{Role: RoleUser}},
		{name: "status", input: UpdateUserInput{Status: StatusDisabled}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := &userRepoStub{user: &User{
				ID:     42,
				Email:  "admin@example.com",
				Role:   RoleAdmin,
				Status: StatusActive,
			}}
			repo := &rpmUserRepoStub{userRepoStub: base}
			svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}}

			input := tt.input
			input.ActorAdminID = 7
			_, err := svc.UpdateUser(context.Background(), 42, &input)

			require.Error(t, err)
			require.Equal(t, "ADMIN_PEER_PROTECTED", infraerrors.Reason(err))
			require.Nil(t, repo.lastUpdated, "peer-admin protected fields must not be persisted")
		})
	}
}

func TestAdminService_UpdateUser_AllowsPeerAdminUnchangedProtectedFields(t *testing.T) {
	base := &userRepoStub{user: &User{
		ID:     42,
		Email:  "admin@example.com",
		Role:   RoleAdmin,
		Status: StatusActive,
	}}
	repo := &rpmUserRepoStub{userRepoStub: base}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}}

	newName := "operations"
	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		Email:        "admin@example.com",
		Role:         RoleAdmin,
		Status:       StatusActive,
		Username:     &newName,
		ActorAdminID: 7,
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, RoleAdmin, updated.Role)
	require.Equal(t, newName, updated.Username)
	require.NotNil(t, repo.lastUpdated)
}

func TestAdminService_UpdateUser_AllowsAdminToChangeOwnLogin(t *testing.T) {
	base := &userRepoStub{user: &User{
		ID:     42,
		Email:  "admin@example.com",
		Role:   RoleAdmin,
		Status: StatusActive,
	}}
	repo := &rpmUserRepoStub{userRepoStub: base}
	svc := &adminServiceImpl{userRepo: repo, redeemCodeRepo: &redeemRepoStub{}}

	updated, err := svc.UpdateUser(context.Background(), 42, &UpdateUserInput{
		Email:        "admin-new@example.com",
		Password:     "new-secure-password",
		ActorAdminID: 42,
	})

	require.NoError(t, err)
	require.NotNil(t, updated)
	require.Equal(t, "admin-new@example.com", updated.Email)
	require.NotNil(t, repo.lastUpdated)
}
