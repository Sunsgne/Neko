package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/neko/sdwan/backend/internal/inventory"
	"github.com/neko/sdwan/backend/internal/store"
	"github.com/neko/sdwan/backend/internal/tenant"
)

// Meta carries pagination metadata in responses.
type Meta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

type envelope struct {
	Data any   `json:"data,omitempty"`
	Meta *Meta `json:"meta,omitempty"`
}

type errBody struct {
	Error errDetail `json:"error"`
}

type errDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, envelope{Data: data})
}

func respondList(w http.ResponseWriter, data any, meta Meta) {
	writeJSON(w, http.StatusOK, envelope{Data: data, Meta: &meta})
}

func respondError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errBody{Error: errDetail{Code: code, Message: msg}})
}

// respondServiceError maps domain errors to HTTP responses.
func respondServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		respondError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, store.ErrConflict):
		respondError(w, http.StatusConflict, "conflict", "resource already exists")
	case errors.Is(err, tenant.ErrInvalidInput), errors.Is(err, inventory.ErrInvalidInput):
		respondError(w, http.StatusBadRequest, "invalid_input", err.Error())
	default:
		respondError(w, http.StatusInternalServerError, "internal", "internal server error")
	}
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
