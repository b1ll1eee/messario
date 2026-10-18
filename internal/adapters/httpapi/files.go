package httpapi

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/b1ll1eee/nuntio-message/internal/app"
	"github.com/b1ll1eee/nuntio-message/internal/domain"
)

// uploadFile handles multipart POST /api/chats/{id}/files with field "file"
// (and optional "client_msg_id").
func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	m, err := s.readUpload(w, r, app.UserFrom(r.Context()), id, "file")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// readUpload is shared by the user API and the bot API (field "document").
func (s *Server) readUpload(w http.ResponseWriter, r *http.Request, sender *domain.User, chatID int64, field string) (*domain.Message, error) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxFileBytes+1<<20)
	if err := r.ParseMultipartForm(1 << 20); err != nil { // keep 1 MiB in memory, rest on disk
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return nil, domain.Invalid("file exceeds %d MB", s.cfg.MaxFileBytes>>20)
		}
		return nil, domain.Invalid("invalid multipart body: %v", err)
	}
	defer r.MultipartForm.RemoveAll() //nolint:errcheck
	f, hdr, err := r.FormFile(field)
	if err != nil {
		return nil, domain.Invalid("missing %q field", field)
	}
	defer f.Close()
	contentType := hdr.Header.Get("Content-Type")
	if mt, _, err := mime.ParseMediaType(contentType); err == nil {
		contentType = mt
	} else {
		contentType = ""
	}
	return s.app.UploadFile(r.Context(), sender, chatID, hdr.Filename, contentType, hdr.Size, f, r.FormValue("client_msg_id"))
}

// downloadFile streams GET /api/files/{id}. Images and PDFs show inline;
// everything else downloads, with the original name.
func (s *Server) downloadFile(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	f, rc, err := s.app.OpenFile(r.Context(), app.UserFrom(r.Context()), id)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	defer rc.Close()
	disposition := "attachment"
	if strings.HasPrefix(f.Mime, "image/") || f.Mime == "application/pdf" {
		disposition = "inline"
	}
	w.Header().Set("Content-Type", f.Mime)
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size, 10))
	w.Header().Set("Content-Disposition", fmt.Sprintf(`%s; filename*=UTF-8''%s`, disposition, escapeFilename(f.Name)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	_, _ = io.Copy(w, rc)
}

// escapeFilename percent-encodes for RFC 5987 filename*.
func escapeFilename(name string) string {
	var b strings.Builder
	for _, c := range []byte(name) {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '.' || c == '-' || c == '_' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
