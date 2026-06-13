// Package idgen generates sortable, unique identifiers.
package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// New returns a time-prefixed random identifier, e.g. "dev_18f0a1b2c3-9f1e...".
// The time prefix makes IDs roughly sortable by creation time.
func New(prefix string) string {
	ts := time.Now().UTC().UnixMilli()
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	if prefix == "" {
		return fmt.Sprintf("%x%s", ts, hex.EncodeToString(b))
	}
	return fmt.Sprintf("%s_%x%s", prefix, ts, hex.EncodeToString(b))
}
