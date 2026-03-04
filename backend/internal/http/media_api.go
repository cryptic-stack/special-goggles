package httpapi

import (
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	maxMediaUploadBytes = 20 << 20
	maxFilesPerUpload   = 4
)

type mediaUploadResponse struct {
	ID           int64  `json:"id"`
	URL          string `json:"url"`
	ContentType  string `json:"content_type"`
	ByteSize     int64  `json:"byte_size"`
	OriginalName string `json:"original_name"`
}

func handleUploadMedia(deps Dependencies) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, err := loadSessionPrincipal(r.Context(), deps, r)
		if errors.Is(err, errUnauthorized) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		if err != nil {
			deps.Logger.Error("media session lookup failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		if r.Body == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing request body"})
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxMediaUploadBytes)
		if err := r.ParseMultipartForm(maxMediaUploadBytes); err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "payload_too_large"})
				return
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_multipart_form"})
			return
		}

		files := r.MultipartForm.File["file"]
		if len(files) == 0 {
			files = r.MultipartForm.File["files"]
		}
		if len(files) == 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "file_required"})
			return
		}
		if len(files) > maxFilesPerUpload {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "too_many_files"})
			return
		}

		dir := filepath.Join(deps.Config.DataDir, "media", strconv.FormatInt(principal.ActorID, 10))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			deps.Logger.Error("media mkdir failed", "error", err, "dir", dir)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
			return
		}

		attachments := make([]mediaUploadResponse, 0, len(files))
		for _, fileHeader := range files {
			uploaded, err := saveUploadedMedia(r, deps, principal.ActorID, dir, fileHeader)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			attachments = append(attachments, uploaded)
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"items": attachments,
		})
	}
}

func saveUploadedMedia(r *http.Request, deps Dependencies, actorID int64, dir string, fileHeader *multipart.FileHeader) (mediaUploadResponse, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return mediaUploadResponse{}, errors.New("failed_to_open_file")
	}
	defer src.Close()

	sniffBuf := make([]byte, 512)
	n, err := io.ReadFull(src, sniffBuf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return mediaUploadResponse{}, errors.New("failed_to_read_file")
	}
	contentType := strings.TrimSpace(fileHeader.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = http.DetectContentType(sniffBuf[:n])
	}
	if !isAllowedMediaType(contentType) {
		return mediaUploadResponse{}, errors.New("unsupported_media_type")
	}

	token, err := randomToken(12)
	if err != nil {
		return mediaUploadResponse{}, errors.New("failed_to_generate_token")
	}
	ext := mediaExtension(fileHeader.Filename, contentType)
	filename := token + ext
	relativeKey := filepath.ToSlash(strconv.FormatInt(actorID, 10) + "/" + filename)
	targetPath := filepath.Join(dir, filename)

	dst, err := os.Create(targetPath)
	if err != nil {
		return mediaUploadResponse{}, errors.New("failed_to_create_file")
	}
	defer dst.Close()

	if n > 0 {
		if _, err := dst.Write(sniffBuf[:n]); err != nil {
			return mediaUploadResponse{}, errors.New("failed_to_write_file")
		}
	}
	written, err := io.Copy(dst, src)
	if err != nil {
		return mediaUploadResponse{}, errors.New("failed_to_write_file")
	}
	totalSize := int64(n) + written

	var attachmentID int64
	err = deps.PG.QueryRow(r.Context(), `
INSERT INTO media_attachments (actor_id, storage_key, content_type, byte_size, original_name)
VALUES ($1, $2, $3, $4, $5)
RETURNING id
`,
		actorID,
		relativeKey,
		contentType,
		totalSize,
		strings.TrimSpace(fileHeader.Filename),
	).Scan(&attachmentID)
	if err != nil {
		_ = os.Remove(targetPath)
		return mediaUploadResponse{}, errors.New("failed_to_store_metadata")
	}

	return mediaUploadResponse{
		ID:           attachmentID,
		URL:          mediaURL(deps.Config.AppBaseURL, relativeKey),
		ContentType:  contentType,
		ByteSize:     totalSize,
		OriginalName: strings.TrimSpace(fileHeader.Filename),
	}, nil
}

func isAllowedMediaType(contentType string) bool {
	contentType = strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return true
	case strings.HasPrefix(contentType, "video/"):
		return true
	case strings.HasPrefix(contentType, "audio/"):
		return true
	case contentType == "application/pdf":
		return true
	default:
		return false
	}
}

func mediaExtension(filename, contentType string) string {
	ext := strings.ToLower(filepath.Ext(strings.TrimSpace(filename)))
	if ext != "" && len(ext) <= 10 {
		return ext
	}
	if guessed, _ := mime.ExtensionsByType(strings.TrimSpace(strings.Split(contentType, ";")[0])); len(guessed) > 0 {
		if len(guessed[0]) <= 10 {
			return strings.ToLower(guessed[0])
		}
	}
	return ".bin"
}

func mediaURL(baseURL, storageKey string) string {
	return strings.TrimRight(baseURL, "/") + "/media/" + strings.TrimLeft(filepath.ToSlash(storageKey), "/")
}
