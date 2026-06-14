package httpapi

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/neko/sdwan/backend/internal/audit"
)

// record writes an audit entry for a mutating action, if a recorder is wired.
func (s *Server) record(ctx context.Context, action, objectType, objectID string, after map[string]string) {
	if s.audit == nil {
		return
	}
	p, _ := principalFrom(ctx)
	id := "aud_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if s.idgen != nil {
		id = s.idgen("aud")
	}
	_ = s.audit.Record(ctx, audit.Entry{
		ID:         id,
		TenantID:   tenantFrom(ctx),
		ActorID:    p.Email,
		Action:     action,
		ObjectType: objectType,
		ObjectID:   objectID,
		After:      after,
		At:         time.Now().UTC(),
	})
}

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	if s.audit == nil {
		respondList(w, []any{}, Meta{Page: 1, PageSize: 20, Total: 0})
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	entries, err := s.audit.List(r.Context(), tenantFrom(r.Context()), limit)
	if err != nil {
		respondServiceError(w, err)
		return
	}
	respondList(w, entries, Meta{Page: 1, PageSize: len(entries), Total: len(entries)})
}
