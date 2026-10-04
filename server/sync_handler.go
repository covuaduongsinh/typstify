package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"looz.ws/typstify/service/vpssync"
)

// Endpoint đồng bộ file cho client desktop đẩy project lên VPS (xem
// service/vpssync). Mọi đường dẫn đều đi qua resolveInRoot; các thư mục
// metadata (.git, .typstify, ...) bị từ chối ở cả pull/push/delete.

// handleSyncManifest trả về danh sách file của project đang mở.
func (s *Server) handleSyncManifest(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	items, err := vpssync.ScanManifest(root)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// handleSyncPull trả về nội dung thô của một file.
func (s *Server) handleSyncPull(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	p, rel, ok := syncTargetPath(w, root, r.URL.Query().Get("path"))
	if !ok {
		return
	}
	info, err := os.Stat(p)
	if err != nil || info.IsDir() {
		writeError(w, http.StatusNotFound, "không tìm thấy tệp "+rel)
		return
	}

	data, err := os.ReadFile(p)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(data)
}

// handleSyncPush ghi nội dung thô của body vào đường dẫn path. Ghi qua file
// tạm rồi rename để không để lại file dở dang nếu ngắt giữa chừng. Nội dung
// được lưu nguyên, không qua repairTypstContent, để hash phía server luôn khớp
// với hash client đã tính.
func (s *Server) handleSyncPush(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	p, rel, ok := syncTargetPath(w, root, r.URL.Query().Get("path"))
	if !ok {
		return
	}

	// withBodyLimit đã giới hạn body; vượt quá sẽ lỗi 413 thay vì ghi cụt.
	data, err := io.ReadAll(r.Body)
	if err != nil {
		writeBodyError(w, err)
		return
	}

	if info, err := os.Lstat(p); err == nil && info.IsDir() {
		writeError(w, http.StatusConflict, rel+" là một thư mục")
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := writeFileAtomic(p, data); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type syncDeleteRequest struct {
	Path string `json:"path"`
}

// handleSyncDelete xoá một file. Xoá file đã không còn là thành công (idempotent).
func (s *Server) handleSyncDelete(w http.ResponseWriter, r *http.Request) {
	root, err := s.projectRoot()
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}

	var req syncDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeBodyError(w, err)
		return
	}

	p, rel, ok := syncTargetPath(w, root, req.Path)
	if !ok {
		return
	}
	info, err := os.Lstat(p)
	if os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, rel+" là thư mục, chỉ xoá được tệp")
		return
	}
	if err := os.Remove(p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// syncTargetPath giải quyết đường dẫn đồng bộ về tuyệt đối trong root và từ
// chối gốc project cùng thư mục metadata. Khi không hợp lệ, nó tự ghi lỗi và
// trả ok=false.
func syncTargetPath(w http.ResponseWriter, root, raw string) (abs, rel string, ok bool) {
	abs, err := resolveInRoot(root, raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return "", "", false
	}
	rel, err = relPath(root, abs)
	if err != nil || rel == "." || vpssync.IsDefaultIgnored(rel) {
		writeError(w, http.StatusBadRequest, "đường dẫn không được phép đồng bộ: "+raw)
		return "", "", false
	}
	return abs, rel, true
}

// writeFileAtomic ghi data vào p qua file tạm cùng thư mục rồi rename.
func writeFileAtomic(p string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(p), "."+filepath.Base(p)+".sync-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op sau khi rename thành công

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, 0644); err != nil {
		return err
	}
	return os.Rename(tmpName, p)
}
