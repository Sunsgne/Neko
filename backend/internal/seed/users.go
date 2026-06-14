package seed

import (
	"context"

	"github.com/neko/sdwan/backend/internal/users"
)

// Users seeds the platform operator account plus demo tenant accounts.
//
// adminEmail/adminPassword override the operator credentials when provided
// (used by production deploys); otherwise demo defaults are used:
//
//	operator: admin@neko.io     / neko12345   (平台运营，可见全部租户)
//	tenant:   ops@acme-corp.com / acme12345   (仅 Acme Corp 租户)
func Users(ctx context.Context, repo users.Repository, adminEmail, adminPassword string) {
	email := adminEmail
	if email == "" {
		email = "admin@neko.io"
	}
	pass := adminPassword
	if pass == "" {
		pass = "neko12345"
	}
	_ = repo.Add(ctx, users.User{
		ID:          "usr_admin",
		Email:       email,
		DisplayName: "平台运营",
		IsOperator:  true,
	}, pass)

	_ = repo.Add(ctx, users.User{
		ID:          "usr_acme",
		Email:       "ops@acme-corp.com",
		DisplayName: "Acme 运维",
		TenantID:    "ten_acme",
	}, "acme12345")
}
