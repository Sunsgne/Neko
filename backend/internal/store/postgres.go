package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is a pgx-backed Store implementation (T1.1). Multi-tenant
// isolation is enforced at the application layer here and, when migration
// 0002 is applied, also by PostgreSQL RLS (T1.3) via the app.tenant_id GUC.
type PostgresStore struct {
	pool    *pgxpool.Pool
	tenants *pgTenantRepo
	devices *pgDeviceRepo
}

// OpenPostgres connects to PostgreSQL and verifies connectivity.
func OpenPostgres(ctx context.Context, dsn string) (*PostgresStore, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &PostgresStore{
		pool:    pool,
		tenants: &pgTenantRepo{pool: pool},
		devices: &pgDeviceRepo{pool: pool},
	}, nil
}

// Close releases the connection pool.
func (s *PostgresStore) Close() { s.pool.Close() }

// Migrate applies pending schema migrations.
func (s *PostgresStore) Migrate(ctx context.Context) error { return Migrate(ctx, s.pool) }

func (s *PostgresStore) Tenants() TenantRepository { return s.tenants }
func (s *PostgresStore) Devices() DeviceRepository { return s.devices }

func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	// 23505 = unique_violation
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
		return ErrConflict
	}
	return err
}

type pgTenantRepo struct{ pool *pgxpool.Pool }

func (r *pgTenantRepo) Create(ctx context.Context, t *Tenant) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO tenants (id, name, slug, status, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		t.ID, t.Name, t.Slug, string(t.Status), t.CreatedAt, t.UpdatedAt)
	return mapPgError(err)
}

func (r *pgTenantRepo) Get(ctx context.Context, id string) (*Tenant, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, status, created_at, updated_at FROM tenants WHERE id=$1`, id)
	var t Tenant
	var status string
	if err := row.Scan(&t.ID, &t.Name, &t.Slug, &status, &t.CreatedAt, &t.UpdatedAt); err != nil {
		return nil, mapPgError(err)
	}
	t.Status = TenantStatus(status)
	return &t, nil
}

func (r *pgTenantRepo) List(ctx context.Context, page Page) ([]*Tenant, int, error) {
	page = page.Normalize()
	var total int
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&total); err != nil {
		return nil, 0, mapPgError(err)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, name, slug, status, created_at, updated_at FROM tenants
		 ORDER BY created_at ASC LIMIT $1 OFFSET $2`, page.Size, page.Offset())
	if err != nil {
		return nil, 0, mapPgError(err)
	}
	defer rows.Close()
	var out []*Tenant
	for rows.Next() {
		var t Tenant
		var status string
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &status, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, err
		}
		t.Status = TenantStatus(status)
		out = append(out, &t)
	}
	return out, total, rows.Err()
}

type pgDeviceRepo struct{ pool *pgxpool.Pool }

func (r *pgDeviceRepo) Create(ctx context.Context, d *Device) error {
	caps, err := marshalCaps(d.Capabilities)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO devices (id, tenant_id, name, mgmt_address, platform, model, serial, trust_state, capabilities, last_seen_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		d.ID, d.TenantID, d.Name, d.MgmtAddress, string(d.Platform), d.Model, d.Serial,
		string(d.TrustState), caps, d.LastSeenAt, d.CreatedAt, d.UpdatedAt)
	return mapPgError(err)
}

func (r *pgDeviceRepo) Get(ctx context.Context, tenantID, id string) (*Device, error) {
	q := `SELECT id, tenant_id, name, mgmt_address, platform, model, serial, trust_state, capabilities, last_seen_at, created_at, updated_at
	      FROM devices WHERE id=$1`
	args := []any{id}
	if tenantID != "" {
		q += ` AND tenant_id=$2`
		args = append(args, tenantID)
	}
	return scanDevice(r.pool.QueryRow(ctx, q, args...))
}

func (r *pgDeviceRepo) List(ctx context.Context, tenantID string, page Page) ([]*Device, int, error) {
	page = page.Normalize()
	countQ := `SELECT count(*) FROM devices`
	listQ := `SELECT id, tenant_id, name, mgmt_address, platform, model, serial, trust_state, capabilities, last_seen_at, created_at, updated_at FROM devices`
	var args []any
	if tenantID != "" {
		countQ += ` WHERE tenant_id=$1`
		listQ += ` WHERE tenant_id=$1`
		args = append(args, tenantID)
	}
	var total int
	if err := r.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, mapPgError(err)
	}
	listQ += fmt.Sprintf(` ORDER BY created_at ASC LIMIT $%d OFFSET $%d`, len(args)+1, len(args)+2)
	args = append(args, page.Size, page.Offset())
	rows, err := r.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, 0, mapPgError(err)
	}
	defer rows.Close()
	var out []*Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, d)
	}
	return out, total, rows.Err()
}

func (r *pgDeviceRepo) Update(ctx context.Context, d *Device) error {
	caps, err := marshalCaps(d.Capabilities)
	if err != nil {
		return err
	}
	ct, err := r.pool.Exec(ctx,
		`UPDATE devices SET name=$2, mgmt_address=$3, platform=$4, model=$5, serial=$6,
		 trust_state=$7, capabilities=$8, last_seen_at=$9, updated_at=$10 WHERE id=$1`,
		d.ID, d.Name, d.MgmtAddress, string(d.Platform), d.Model, d.Serial,
		string(d.TrustState), caps, d.LastSeenAt, d.UpdatedAt)
	if err != nil {
		return mapPgError(err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDevice(row rowScanner) (*Device, error) {
	var d Device
	var platform, trust string
	var caps []byte
	if err := row.Scan(&d.ID, &d.TenantID, &d.Name, &d.MgmtAddress, &platform, &d.Model,
		&d.Serial, &trust, &caps, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, mapPgError(err)
	}
	d.Platform = DevicePlatform(platform)
	d.TrustState = TrustState(trust)
	if len(caps) > 0 {
		var cm CapabilityMatrix
		if err := json.Unmarshal(caps, &cm); err == nil {
			d.Capabilities = &cm
		}
	}
	return &d, nil
}

func marshalCaps(c *CapabilityMatrix) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}
