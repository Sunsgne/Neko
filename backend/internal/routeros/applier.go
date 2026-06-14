package routeros

import (
	"context"
	"fmt"

	"github.com/neko/sdwan/backend/internal/configengine"
)

// Applier implements configengine.Applier against a RouterOS device over REST,
// so the platform can push full configuration without logging in.
//
// Statement identity: the engine uses a synthetic Key per (path,key). To map a
// desired statement onto a concrete RouterOS item we match on a per-path key
// field (defaulting to "name", falling back to ".id"). This is sufficient for
// the managed sections, which all carry a natural key.
type Applier struct {
	client   *Client
	sections []string
	keyField map[string]string // path -> attribute used as natural key
}

// NewApplier builds an Applier for a device target. sections defaults to
// ManagedSections when empty.
func NewApplier(t Target, sections []string) *Applier {
	if len(sections) == 0 {
		sections = ManagedSections
	}
	return &Applier{
		client:   NewClient(t),
		sections: sections,
		keyField: defaultKeyFields(),
	}
}

func defaultKeyFields() map[string]string {
	return map[string]string{
		"/ip/address":               "address",
		"/ip/route":                 "dst-address",
		"/ip/firewall/address-list": "address",
		"/interface/vlan":           "name",
		"/interface/bridge":         "name",
		"/interface/wireguard":      "name",
		"/snmp/community":           "name",
		"/ip/dns/static":            "name",
	}
}

func (a *Applier) keyFor(path string) string {
	if k, ok := a.keyField[path]; ok {
		return k
	}
	return "name"
}

// Snapshot reads the managed sections into a configengine.State.
func (a *Applier) Snapshot(ctx context.Context) (configengine.State, error) {
	var sts []configengine.Statement
	for _, path := range a.sections {
		items, err := a.client.List(ctx, path)
		if err != nil {
			// Section may be unsupported on this device/license; skip it.
			continue
		}
		kf := a.keyFor(path)
		for _, it := range items {
			key := stringField(it, kf)
			if key == "" {
				key = stringField(it, ".id")
			}
			if key == "" {
				continue
			}
			sts = append(sts, configengine.Statement{
				Path:       path,
				Key:        key,
				Attributes: toStringMap(it),
			})
		}
	}
	return configengine.State{Statements: sts}, nil
}

// Apply executes the plan via REST CRUD. confirmTimeoutSec is accepted for
// interface compatibility; commit-confirm is provided at a higher level by the
// snapshot+Restore safety net (RouterOS REST has no native commit-confirm).
func (a *Applier) Apply(ctx context.Context, plan configengine.Plan, _ int) error {
	for _, ch := range plan.Changes {
		switch ch.Type {
		case configengine.ChangeAdd:
			attrs := attrsFromChange(ch)
			if err := a.client.Create(ctx, ch.Path, attrs); err != nil {
				return fmt.Errorf("create %s/%s: %w", ch.Path, ch.Key, err)
			}
		case configengine.ChangeUpdate:
			id, err := a.resolveID(ctx, ch.Path, ch.Key)
			if err != nil {
				return err
			}
			if err := a.client.Update(ctx, ch.Path, id, attrsFromChange(ch)); err != nil {
				return fmt.Errorf("update %s/%s: %w", ch.Path, ch.Key, err)
			}
		case configengine.ChangeRemove:
			id, err := a.resolveID(ctx, ch.Path, ch.Key)
			if err != nil {
				return err
			}
			if err := a.client.Delete(ctx, ch.Path, id); err != nil {
				return fmt.Errorf("delete %s/%s: %w", ch.Path, ch.Key, err)
			}
		}
	}
	return nil
}

// Confirm is a no-op for REST (safety provided by Restore).
func (a *Applier) Confirm(_ context.Context) error { return nil }

// Restore re-converges the device toward a previously captured snapshot by
// diffing the current running config against it and applying the reverse plan.
func (a *Applier) Restore(ctx context.Context, snapshot configengine.State) error {
	current, err := a.Snapshot(ctx)
	if err != nil {
		return err
	}
	plan := configengine.ComputeDiff(current, snapshot, configengine.RiskOptions{})
	return a.Apply(ctx, plan, 0)
}

// resolveID looks up the RouterOS .id for an item identified by its natural key.
func (a *Applier) resolveID(ctx context.Context, path, key string) (string, error) {
	items, err := a.client.List(ctx, path)
	if err != nil {
		return "", err
	}
	kf := a.keyFor(path)
	for _, it := range items {
		if stringField(it, kf) == key || stringField(it, ".id") == key {
			if id := stringField(it, ".id"); id != "" {
				return id, nil
			}
		}
	}
	return "", fmt.Errorf("item not found: %s key=%s", path, key)
}

func attrsFromChange(ch configengine.Change) map[string]string {
	out := map[string]string{}
	for _, a := range ch.Attrs {
		if a.New != "" {
			out[a.Attr] = a.New
		}
	}
	return out
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k]; ok {
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func toStringMap(m map[string]any) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		if k == ".id" {
			continue
		}
		out[k] = fmt.Sprintf("%v", v)
	}
	return out
}
