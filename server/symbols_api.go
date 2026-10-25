package server

import (
	"context"
	"net/http"
	"sort"
	"strings"
	"sync"

	"looz.ws/typstify/lsp"
)

// symbolInfo is one Typst symbol for the web symbol picker.
type symbolInfo struct {
	Name     string `json:"name"` // e.g. "arrow.r.long" (without the sym. prefix)
	Char     string `json:"char"`
	Category string `json:"category,omitempty"`
}

// symbolsCache holds tinymist's symbol table: it only changes with the
// tinymist version, so one successful fetch serves the whole process.
var symbolsCache struct {
	sync.Mutex
	list []symbolInfo
}

// parseSymbols reads tinymist.getResources("/symbols"):
// {"symbols": [{"id": "sym.arrow.r", "value": "→", "category": "arrow", "glyph": "<svg…>"}]}.
// The rendered glyph is dropped: the browser draws the character itself.
func parseSymbols(res any) []symbolInfo {
	root, _ := res.(map[string]any)
	items, _ := root["symbols"].([]any)
	out := make([]symbolInfo, 0, len(items))
	for _, v := range items {
		item, _ := v.(map[string]any)
		id, _ := item["id"].(string)
		value, _ := item["value"].(string)
		if !strings.HasPrefix(id, "sym.") || value == "" {
			continue
		}
		cat, _ := item["category"].(string)
		out = append(out, symbolInfo{Name: strings.TrimPrefix(id, "sym."), Char: value, Category: cat})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// handleSymbols lists Typst's symbols (sym.*) for the symbol picker.
func (s *Server) handleSymbols(w http.ResponseWriter, r *http.Request) {
	symbolsCache.Lock()
	defer symbolsCache.Unlock()
	if symbolsCache.list == nil {
		root, err := s.projectRoot()
		if err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		client := lsp.GetLspClient(root, s.appSrv.Settings())
		if client == nil || !client.IsReady() {
			writeError(w, http.StatusServiceUnavailable, "tinymist chưa sẵn sàng")
			return
		}
		res, err := client.ExecuteCommand(context.Background(), "tinymist.getResources", []any{"/symbols"})
		if err != nil {
			writeError(w, http.StatusBadGateway, err.Error())
			return
		}
		list := parseSymbols(res)
		if len(list) == 0 {
			writeError(w, http.StatusBadGateway, "tinymist không trả về ký hiệu")
			return
		}
		symbolsCache.list = list
	}
	w.Header().Set("Cache-Control", "private, max-age=86400")
	writeJSON(w, http.StatusOK, symbolsCache.list)
}
