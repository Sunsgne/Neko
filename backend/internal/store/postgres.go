package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	creds   *pgCredentialRepo
	alerts  *pgAlertRepo
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
		creds:   &pgCredentialRepo{pool: pool},
		alerts:  &pgAlertRepo{pool: pool},
	}, nil
}

// Close releases the connection pool.
func (s *PostgresStore) Close() { s.pool.Close() }

// Migrate applies pending schema migrations.
func (s *PostgresStore) Migrate(ctx context.Context) error { return Migrate(ctx, s.pool) }

func (s *PostgresStore) Tenants() TenantRepository         { return s.tenants }
func (s *PostgresStore) Devices() DeviceRepository         { return s.devices }
func (s *PostgresStore) Credentials() CredentialRepository { return s.creds }
func (s *PostgresStore) Alerts() AlertRepository           { return s.alerts }

type pgAlertRepo struct{ pool *pgxpool.Pool }

func (r *pgAlertRepo) Fire(ctx context.Context, a Alert) (*Alert, bool, error) {
	// Return existing open alert if present.
	var id string
	err := r.pool.QueryRow(ctx,
		`SELECT id FROM alerts WHERE device_id=$1 AND code=$2 AND state='firing' LIMIT 1`,
		nullable(a.DeviceID), a.Code).Scan(&id)
	if err == nil {
		a.ID = id
		a.State = "firing"
		return &a, false, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, mapPgError(err)
	}
	if a.FiredAt.IsZero() {
		a.FiredAt = time.Now().UTC()
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO alerts (id, tenant_id, device_id, code, severity, title, detail, state, fired_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'firing',$8)`,
		a.ID, nullable(a.TenantID), nullable(a.DeviceID), a.Code, a.Severity, a.Title, a.Detail, a.FiredAt)
	if err != nil {
		return nil, false, mapPgError(err)
	}
	a.State = "firing"
	return &a, true, nil
}

func (r *pgAlertRepo) Resolve(ctx context.Context, deviceID, code string, at time.Time) (bool, error) {
	ct, err := r.pool.Exec(ctx,
		`UPDATE alerts SET state='resolved', resolved_at=$3
		 WHERE device_id=$1 AND code=$2 AND state='firing'`,
		nullable(deviceID), code, at)
	if err != nil {
		return false, mapPgError(err)
	}
	return ct.RowsAffected() > 0, nil
}

func (r *pgAlertRepo) List(ctx context.Context, tenantID string, limit int) ([]*Alert, error) {
	if limit <= 0 {
		limit = 200
	}
	q := `SELECT id, coalesce(tenant_id,''), coalesce(device_id,''), code, severity, title, detail, state, fired_at, resolved_at FROM alerts`
	var args []any
	if tenantID != "" {
		q += ` WHERE tenant_id=$1`
		args = append(args, tenantID)
	}
	q += ` ORDER BY (state='firing') DESC, fired_at DESC LIMIT ` + strconv.Itoa(limit)
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, mapPgError(err)
	}
	defer rows.Close()
	var out []*Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.TenantID, &a.DeviceID, &a.Code, &a.Severity, &a.Title, &a.Detail, &a.State, &a.FiredAt, &a.ResolvedAt); err != nil {
			return nil, err
		}
		out = append(out, &a)
	}
	return out, rows.Err()
}

type pgCredentialRepo struct{ pool *pgxpool.Pool }

func (r *pgCredentialRepo) Put(ctx context.Context, c Credential) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO device_credentials (device_id, kind, encrypted_blob)
		 VALUES ($1,$2,$3)
		 ON CONFLICT (device_id) DO UPDATE SET kind=$2, encrypted_blob=$3`,
		c.DeviceID, c.Kind, []byte(c.Sealed))
	return mapPgError(err)
}

func (r *pgCredentialRepo) Get(ctx context.Context, deviceID string) (*Credential, error) {
	var c Credential
	var blob []byte
	err := r.pool.QueryRow(ctx,
		`SELECT device_id, kind, encrypted_blob FROM device_credentials WHERE device_id=$1`, deviceID).
		Scan(&c.DeviceID, &c.Kind, &blob)
	if err != nil {
		return nil, mapPgError(err)
	}
	c.Sealed = string(blob)
	return &c, nil
}

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
	role := d.Role
	if role == "" {
		role = RoleCPE
	}
	status, err := marshalStatus(d.Status)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO devices (id, tenant_id, name, mgmt_address, role, region, platform, model, serial, trust_state, capabilities, status, enrolled, last_seen_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		d.ID, nullable(d.TenantID), d.Name, d.MgmtAddress, string(role), d.Region, string(d.Platform), d.Model, d.Serial,
		string(d.TrustState), caps, status, d.Enrolled, d.LastSeenAt, d.CreatedAt, d.UpdatedAt)
	return mapPgError(err)
}

func (r *pgDeviceRepo) Get(ctx context.Context, tenantID, id string) (*Device, error) {
	q := `SELECT id, tenant_id, name, mgmt_address, role, region, platform, model, serial, trust_state, capabilities, status, enrolled, last_seen_at, created_at, updated_at
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
	listQ := `SELECT id, tenant_id, name, mgmt_address, role, region, platform, model, serial, trust_state, capabilities, status, enrolled, last_seen_at, created_at, updated_at FROM devices`
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
	role := d.Role
	if role == "" {
		role = RoleCPE
	}
	status, err := marshalStatus(d.Status)
	if err != nil {
		return err
	}
	ct, err := r.pool.Exec(ctx,
		`UPDATE devices SET name=$2, mgmt_address=$3, role=$4, region=$5, platform=$6, model=$7, serial=$8,
		 trust_state=$9, capabilities=$10, status=$11, enrolled=$12, last_seen_at=$13, updated_at=$14 WHERE id=$1`,
		d.ID, d.Name, d.MgmtAddress, string(role), d.Region, string(d.Platform), d.Model, d.Serial,
		string(d.TrustState), caps, status, d.Enrolled, d.LastSeenAt, d.UpdatedAt)
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
	var platform, trust, role string
	var tenantID *string
	var caps, status []byte
	if err := row.Scan(&d.ID, &tenantID, &d.Name, &d.MgmtAddress, &role, &d.Region, &platform, &d.Model,
		&d.Serial, &trust, &caps, &status, &d.Enrolled, &d.LastSeenAt, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, mapPgError(err)
	}
	if tenantID != nil {
		d.TenantID = *tenantID
	}
	d.Role = DeviceRole(role)
	d.Platform = DevicePlatform(platform)
	d.TrustState = TrustState(trust)
	if len(caps) > 0 {
		var cm CapabilityMatrix
		if err := json.Unmarshal(caps, &cm); err == nil {
			d.Capabilities = &cm
		}
	}
	if len(status) > 0 {
		var st DeviceStatus
		if err := json.Unmarshal(status, &st); err == nil {
			d.Status = &st
		}
	}
	return &d, nil
}

func marshalStatus(s *DeviceStatus) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// nullable converts an empty string to a SQL NULL so platform-owned devices
// (no tenant) satisfy the nullable tenant_id column instead of violating the
// foreign key with an empty string.
func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func marshalCaps(c *CapabilityMatrix) ([]byte, error) {
	if c == nil {
		return nil, nil
	}
	return json.Marshal(c)
}
