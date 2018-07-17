package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// ErrorResponse struct
type ErrorResponse struct {
	Error string `json:"error"`
}

// UploadResponse struct
type UploadResponse struct {
	Hash string `json:"hash"`
}

// Handler struct
type Handler struct {
	App *Application
}

// ServeHTTP method is application router
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParts := getPathParts(r.URL.Path)
	// for simplicity we don't check ip from headers (X-FORWARDED-FOR, X-REAL-IP)
	ip := strings.Split(r.RemoteAddr, ":")[0]

	l := len(pathParts)

	if l < 1 || pathParts[0] != "files" || l > 2 {
		// not found
		h.renderError(w, http.StatusNotFound, "NOT_FOUND")
		return
	}

	// check connections limit
	allowed := h.App.RateLimit.AddConnection(ip)
	defer h.App.RateLimit.RemoveConnection(ip)

	if !allowed {
		h.renderError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
		return
	}

	// file download
	if r.Method == "GET" && l == 2 {
		// check rps
		if !h.App.RateLimit.CheckRPS("download") {
			h.renderError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
			return
		}

		h.downloadFile(w, r, ip, pathParts[1])
		return
	}

	// file upload
	if r.Method == "POST" && l == 1 {
		// check rps
		if !h.App.RateLimit.CheckRPS("upload") {
			h.renderError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
			return
		}

		h.uploadFile(w, r, ip)
		return
	}

	// file removing
	if r.Method == "DELETE" && l == 2 {
		// check rps
		if !h.App.RateLimit.CheckRPS("remove") {
			h.renderError(w, http.StatusTooManyRequests, "TOO_MANY_REQUESTS")
			return
		}

		h.removeFile(w, r, pathParts[1])
		return
	}

	// here we can handle some other routes if we would need

	// not found
	h.renderError(w, http.StatusNotFound, "NOT_FOUND")
}

func (h *Handler) uploadFile(w http.ResponseWriter, r *http.Request, ip string) {
	// tell client that request is too large. it prevents file upload
	if r.ContentLength > h.App.Config.Storage.MaxSize {
		h.renderError(w, http.StatusExpectationFailed, "REQUEST_TOO_LARGE")
		return
	}

	if !h.App.RateLimit.CheckBandwidth("upload", ip, r.ContentLength) {
		h.renderError(w, http.StatusForbidden, "BYTE_LIMIT_REACHED")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, h.App.Config.Storage.MaxSize)

	err := r.ParseMultipartForm(h.App.Config.Storage.MaxSize)
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "BAD_REQUEST")
		return
	}

	f, _, err := r.FormFile("file")
	if err != nil {
		h.renderError(w, http.StatusBadRequest, "BAD_FILE")
		return
	}
	defer f.Close()

	bytesData, err := ioutil.ReadAll(f)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
		return
	}

	hash, err := getSHA256Sum(bytes.NewBuffer(bytesData))
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
		return
	}

	// check hashes
	sha256Hash := r.FormValue("sha256")
	if sha256Hash != "" && sha256Hash != hash {
		h.renderError(w, http.StatusBadRequest, "BAD_SHA256")
		return
	}

	// check sha1 hash if it has been sent
	sha1Hash := r.FormValue("sha1")
	if sha1Hash != "" {
		fileHash, err := getSHA1Sum(bytes.NewBuffer(bytesData))
		if err != nil {
			h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
			return
		}

		if fileHash != sha1Hash {
			h.renderError(w, http.StatusBadRequest, "BAD_SHA1")
			return
		}
	}

	// check md5 hash if it has been sent
	md5Hash := r.FormValue("md5")
	if md5Hash != "" {
		fileHash, err := getMd5Sum(bytes.NewBuffer(bytesData))
		if err != nil {
			h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
			return
		}

		if fileHash != md5Hash {
			h.renderError(w, http.StatusBadRequest, "BAD_MD5")
			return
		}
	}

	// make hash unique
	uniqHash := hash + "-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.Itoa(rand.Intn(999999))

	// @todo place precallback here

	size, err := h.App.Storage.CreateFile(uniqHash, bytes.NewBuffer(bytesData))
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
		return
	}

	createdAt := time.Now()

	// @todo place postcallback here

	// save meta data to redis
	go h.App.Redis.SaveFileMeta(&FileMeta{
		Hash:      hash,
		Size:      size,
		CreatedAt: &createdAt,
	})

	// render response
	data := UploadResponse{Hash: uniqHash}

	res, err := json.Marshal(data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(res)
}

func (h *Handler) downloadFile(w http.ResponseWriter, r *http.Request, ip, hash string) {
	fileName, ok := h.App.Storage.GetFile(hash)
	if !ok {
		// file not found
		h.renderError(w, http.StatusNotFound, "FILE_NOT_FOUND")
		return
	}

	f, err := os.Open(fileName)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
		return
	}
	defer f.Close()

	bytesData, err := ioutil.ReadAll(f)
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
		return
	}

	if !h.App.RateLimit.CheckBandwidth("download", ip, int64(len(bytesData))) {
		h.renderError(w, http.StatusForbidden, "BYTE_LIMIT_REACHED")
		return
	}

	hashSHA256, err := getSHA256Sum(bytes.NewBuffer(bytesData))
	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
		return
	}

	if strings.Split(hash, "-")[0] != hashSHA256 {
		// file is corrupted
		// maybe we should remove this file?
		h.renderError(w, http.StatusUnprocessableEntity, "FILE_IS_CORRUPTED")
		return
	}

	// update file donwload score
	go h.App.Redis.IncScore(hashSHA256)

	w.Header().Set("Content-Disposition", "attachment; filename="+hash)

	io.Copy(w, bytes.NewBuffer(bytesData))
}

func (h *Handler) removeFile(w http.ResponseWriter, r *http.Request, hash string) {
	ok, err := h.App.Storage.RemoveFile(hash)
	now := time.Now()

	if err != nil {
		h.renderError(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR")
		return
	}

	if !ok {
		h.renderError(w, http.StatusNotFound, "FILE_NOT_FOUND")
		return
	}

	// update file meta data
	go h.App.Redis.MarkFileAsDeleted(hash, &now)

	// it's ok
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) renderError(w http.ResponseWriter, code int, message string) {
	e := ErrorResponse{Error: message}

	res, err := json.Marshal(e)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(code)
	w.Write(res)
}

// NewHandler func return Handler pointer
func NewHandler(app *Application) *Handler {
	return &Handler{
		App: app,
	}
}

func getPathParts(path string) []string {
	pathParts := strings.Split(path, "/")

	res := make([]string, 0, len(pathParts))

	for _, part := range pathParts {
		if part == "" {
			continue
		}

		if strings.HasPrefix(part, "?") {
			break
		}

		dParts := strings.Split(part, "?")
		res = append(res, dParts[0])

		if len(dParts) > 1 {
			break
		}
	}

	return res
}
