package seed

import (
	"context"

	"github.com/neko/sdwan/backend/internal/users"
)

// Users seeds demo accounts. Credentials are intentionally simple for the demo.
//
//	operator: admin@neko.io     / neko12345   (平台运营，可见全部租户)
//	tenant:   ops@acme-corp.com / acme12345   (仅 Acme Corp 租户)
func Users(ctx context.Context, repo users.Repository) {
	_ = repo.Add(ctx, users.User{
		ID:          "usr_admin",
		Email:       "admin@neko.io",
		DisplayName: "平台运营",
		IsOperator:  true,
	}, "neko12345")

	_ = repo.Add(ctx, users.User{
		ID:          "usr_acme",
		Email:       "ops@acme-corp.com",
		DisplayName: "Acme 运维",
		TenantID:    "ten_acme",
	}, "acme12345")
}
