package httpapi

import (
	"net/http"
	"strconv"

	"github.com/neko/sdwan/backend/internal/accel"
	"github.com/neko/sdwan/backend/internal/routeros"
)

// handleChnroutesStatus reports the cached China prefix table (count + source).
func (s *Server) handleChnroutesStatus(w http.ResponseWriter, _ *http.Request) {
	respondData(w, http.StatusOK, s.chn.Status())
}

type chnroutesRefreshRequest struct {
	URL string `json:"url,omitempty"`
}

// handleChnroutesRefresh (re)downloads the chnroutes2 table from the given URL
// (or the default) and caches it for delivery.
func (s *Server) handleChnroutesRefresh(w http.ResponseWriter, r *http.Request) {
	var req chnroutesRefreshRequest
	_ = decodeJSON(r, &req)
	st, err := s.chn.Refresh(r.Context(), req.URL)
	if err != nil {
		respondError(w, http.StatusBadGateway, "chnroutes_fetch_failed", err.Error())
		return
	}
	s.record(r.Context(), "chnroutes_refresh", "chnroutes", st.URL, map[string]string{"count": strconv.Itoa(st.Count)})
	respondData(w, http.StatusOK, st)
}

// chinaSplitRequest delivers (or previews) route-table based 国内外加速.
type chinaSplitRequest struct {
	WANGateway      string `json:"wan_gateway"`
	OverseasGateway string `json:"overseas_gateway"`
	RoutingTable    string `json:"routing_table,omitempty"`
	Distance        int    `json:"distance,omitempty"`
	DryRun          bool   `json:"dry_run"`
	Username        string `json:"username,omitempty"`
	Password        string `json:"password,omitempty"`
}

// handleChinaSplit builds a RouterOS script routing China prefixes via the
// local WAN gateway and 0.0.0.0/1 + 128.0.0.0/1 via the overseas tunnel, then
// either previews it (dry_run) or installs+runs it on the device in one shot.
func (s *Server) handleChinaSplit(w http.ResponseWriter, r *http.Request) {
	dev, err := s.inventory.Get(r.Context(), tenantFrom(r.Context()), r.PathValue("id"))
	if err != nil {
		respondServiceError(w, err)
		return
	}
	var req chinaSplitRequest
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid_json", "request body is not valid JSON")
		return
	}

	// Ensure the China table is loaded (lazy-fetch from the default source).
	if _, err := s.chn.EnsureLoaded(r.Context()); err != nil {
		respondError(w, http.StatusBadGateway, "chnroutes_unavailable",
			"国内路由表未加载，请先在「国内外加速」刷新路由表："+err.Error())
		return
	}
	prefixes := s.chn.Prefixes()

	script, count, err := accel.BuildChinaSplitScript(prefixes, accel.ChinaSplitParams{
		WANGateway:      req.WANGateway,
		OverseasGateway: req.OverseasGateway,
		RoutingTable:    req.RoutingTable,
		Distance:        req.Distance,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid_profile", err.Error())
		return
	}

	if req.DryRun {
		respondData(w, http.StatusOK, map[string]any{
			"dry_run":         true,
			"script":          script,
			"route_count":     count,
			"domestic_count":  len(prefixes),
			"overseas_halves": accel.OverseasHalves,
		})
		return
	}

	client := routeros.NewClient(routeros.Target{
		Address:  dev.MgmtAddress,
		Username: req.Username,
		Secret:   req.Password,
	})
	out, err := client.RunScript(r.Context(), "neko-china-split", script)
	if err != nil {
		respondData(w, http.StatusOK, map[string]any{
			"status":      "failed",
			"route_count": count,
			"error":       err.Error(),
			"output":      out,
		})
		return
	}
	s.record(r.Context(), "china_split_deliver", "device", dev.ID, map[string]string{
		"routes": strconv.Itoa(count),
	})
	respondData(w, http.StatusOK, map[string]any{
		"status":      "delivered",
		"route_count": count,
		"output":      out,
	})
}
