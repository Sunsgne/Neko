package routeros

import (
	"context"
	"errors"
)

// ErrUnreachable indicates the device could not be contacted.
var ErrUnreachable = errors.New("device unreachable")

// Target describes how to reach a device for fact collection.
type Target struct {
	Address  string
	Username string
	// Secret is resolved from encrypted storage by the caller; never logged.
	Secret string
	UseTLS bool
}

// Collector gathers raw facts from a RouterOS device. Implementations:
//   - RestCollector: talks to the RouterOS v7 REST API (real devices).
//   - StaticCollector: returns canned facts (tests / demos / dry-run).
type Collector interface {
	Collect(ctx context.Context, t Target) (*DeviceFacts, error)
}

// StaticCollector returns predefined facts, useful for tests and demos.
type StaticCollector struct {
	Facts *DeviceFacts
	Err   error
}

// Collect implements Collector.
func (s StaticCollector) Collect(_ context.Context, _ Target) (*DeviceFacts, error) {
	if s.Err != nil {
		return nil, s.Err
	}
	return s.Facts, nil
}
