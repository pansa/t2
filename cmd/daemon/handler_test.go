package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestGetPathParts(t *testing.T) {
	cases := []struct {
		path string
		res  []string
	}{
		{
			path: "",
			res:  []string{},
		},
		{
			path: "/",
			res:  []string{},
		},
		{
			path: "/files",
			res:  []string{"files"},
		},
		{
			path: "//files",
			res:  []string{"files"},
		},
		{
			path: "/files/",
			res:  []string{"files"},
		},
		{
			path: "/files/file",
			res:  []string{"files", "file"},
		},
		{
			path: "/files/file/",
			res:  []string{"files", "file"},
		},
		{
			path: "/files/file?file2",
			res:  []string{"files", "file"},
		},
		{
			path: "/files/file/?file2=ok",
			res:  []string{"files", "file"},
		},
	}

	for _, tc := range cases {
		res := getPathParts(tc.path)

		if !reflect.DeepEqual(res, tc.res) {
			t.Errorf("Res must be %v but got %v\n", tc.res, res)
		}
	}
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(&Application{})

	if h == nil {
		t.Errorf("Handler must not be nil")
	}
}

func TestHandlerRenderError(t *testing.T) {
	h := NewHandler(&Application{})

	cases := []struct {
		code    int
		message string
	}{
		{
			code:    500,
			message: "Internal server error",
		},
		{
			code:    404,
			message: "Not found",
		},
		{
			code: 403,
		},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()

		h.renderError(w, tc.code, tc.message)

		if w.Code != tc.code {
			t.Errorf("Code must be %d but got %d\n", tc.code, w.Code)
		}

		contentType := w.Header().Get("Content-type")
		if contentType != "application/json" {
			t.Errorf("Content-type must be %s but got %s\n", "application/json", contentType)
		}

		b := ErrorResponse{}
		json.Unmarshal(w.Body.Bytes(), &b)

		if b.Error != tc.message {
			t.Errorf("Error message must be %s but got %s\n", tc.message, b.Error)
		}
	}
}

func TestHandlerRemoveFile(t *testing.T) {
	cfg, _ := NewConfig("mocks/config/full.json")

	h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

	cases := []struct {
		file *FileMeta
		data []byte
		hash string
		code int
	}{
		{
			file: &FileMeta{
				Hash: "example",
				Size: 128,
			},
			data: []byte(strings.Repeat("a", 128)),
			hash: "example",
			code: 204,
		},
		{
			hash: "example",
			code: 404,
		},
		{
			hash: "example",
			code: 429,
		},
	}

	// flush redis db
	conn := h.App.Redis.Get()
	conn.Do("FLUSHDB")
	conn.Close()

	for _, tc := range cases {
		if tc.data != nil {
			h.App.Storage.CreateFile(tc.hash, bytes.NewBuffer(tc.data))
		}

		if tc.file != nil {
			if tc.file.CreatedAt == nil {
				now := time.Now()
				tc.file.CreatedAt = &now
			}

			h.App.Redis.SaveFileMeta(tc.file)
		}

		w := httptest.NewRecorder()
		r, _ := http.NewRequest("DELETE", "/files/"+tc.hash, nil)

		h.ServeHTTP(w, r)

		if w.Code != tc.code {
			t.Errorf("Code must be %d but got %d\n", tc.code, w.Code)
		}
	}

}

func TestHandlerUploadFile(t *testing.T) {
	cases := []struct {
		code       int
		errMessage string
		file       string
		data       map[string]string
	}{
		{
			code:       400,
			errMessage: "BAD_REQUEST",
		},
		{
			data: map[string]string{
				"sha1": "example",
			},
			code:       400,
			errMessage: "BAD_FILE",
		},
		{
			file: "mocks/files/large.txt",
			data: map[string]string{
				"sha1": "example",
			},
			code:       417,
			errMessage: "REQUEST_TOO_LARGE",
		},
		{
			file: "mocks/files/small.txt",
			data: map[string]string{
				"sha256": "example",
			},
			code:       400,
			errMessage: "BAD_SHA256",
		},
		{
			file: "mocks/files/small.txt",
			data: map[string]string{
				"sha1": "example",
			},
			code:       400,
			errMessage: "BAD_SHA1",
		},
		{
			file: "mocks/files/small.txt",
			data: map[string]string{
				"md5": "example",
			},
			code:       400,
			errMessage: "BAD_MD5",
		},
		{
			file: "mocks/files/small.txt",
			data: map[string]string{
				"sha256": "b4373779db9de9f4782f1d878c5468b24c2d8110d3b322602c0322f486223f0c",
			},
			code: 200,
		},
		{
			file: "mocks/files/small.txt",
			data: map[string]string{
				"sha1": "30209f556027193b730e3c8ea8c4f581234fcdef",
			},
			code: 200,
		},
		{
			file: "mocks/files/small.txt",
			data: map[string]string{
				"md5": "faba42af9c66e079f12e1f160b34744c",
			},
			code: 200,
		},
	}

	for _, tc := range cases {
		cfg, _ := NewConfig("mocks/config/full.json")

		h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

		// before each test we should flush db
		conn := h.App.Redis.Get()
		conn.Do("FLUSHDB")
		conn.Close()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		for key, val := range tc.data {
			writer.WriteField(key, val)
		}

		if tc.file != "" {
			part, err := writer.CreateFormFile("file", filepath.Base(tc.file))
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
				continue
			}

			file, err := os.Open(tc.file)
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
				continue
			}

			_, err = io.Copy(part, file)
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
				continue
			}

			err = file.Close()
		}

		writer.Close()

		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "/files/", body)
		if tc.data != nil || tc.file != "" {
			r.Header.Set("Content-Type", writer.FormDataContentType())
		}

		h.ServeHTTP(w, r)

		if w.Code != tc.code {
			t.Errorf("Code must be %d but got %d\n", tc.code, w.Code)
		}

		if tc.code >= 400 {
			errResp := ErrorResponse{}

			json.Unmarshal(w.Body.Bytes(), &errResp)
			if errResp.Error != tc.errMessage {
				t.Errorf("Error message must be %v but got %v\n", tc.errMessage, errResp.Error)
			}
		}

	}
}

func TestHandlerUploadFileBandwidth(t *testing.T) {
	cfg, _ := NewConfig("mocks/config/full.json")

	// quick fix
	// update max size for bandwodth test
	cfg.Storage.MaxSize = cfg.RateLimit.Bandwidth.Upload + 1024

	h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

	// before test we should flush db
	conn := h.App.Redis.Get()
	conn.Do("FLUSHDB")
	conn.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("example", strings.Repeat("a", int(cfg.RateLimit.Bandwidth.Upload+1)))

	writer.Close()

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("POST", "/files/", body)
	r.Header.Set("Content-Type", writer.FormDataContentType())

	h.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Errorf("Code must be %d but got %d\n", 403, w.Code)
	}

	errResp := ErrorResponse{}

	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "BYTE_LIMIT_REACHED" {
		t.Errorf("Error message must be %v but got %v\n", "BYTE_LIMIT_REACHED", errResp.Error)
	}

}

func TestHandlerUploadRPS(t *testing.T) {
	cfg, _ := NewConfig("mocks/config/full.json")

	h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

	// before test we should flush db
	conn := h.App.Redis.Get()
	conn.Do("FLUSHDB")
	conn.Close()

	for i := 0; i < 3; i++ {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		writer.WriteField("example", strings.Repeat("a", int(cfg.RateLimit.Bandwidth.Upload+1)))

		writer.Close()

		w := httptest.NewRecorder()
		r, _ := http.NewRequest("POST", "/files/", body)
		r.Header.Set("Content-Type", writer.FormDataContentType())

		h.ServeHTTP(w, r)

		if i == 2 {
			if w.Code != 429 {
				t.Errorf("Code must be %d but got %d\n", 429, w.Code)
			}

			errResp := ErrorResponse{}

			json.Unmarshal(w.Body.Bytes(), &errResp)
			if errResp.Error != "TOO_MANY_REQUESTS" {
				t.Errorf("Error message must be %v but got %v\n", "TOO_MANY_REQUESTS", errResp.Error)
			}
		}
	}

}

func TestHandlerDownloadFile(t *testing.T) {
	cases := []struct {
		name       string
		code       int
		errMessage string
		fileName   string
		valid      bool
	}{
		{
			name:       "example",
			code:       404,
			errMessage: "FILE_NOT_FOUND",
		},
		{
			name:       "example",
			code:       422,
			errMessage: "FILE_IS_CORRUPTED",
			fileName:   "mocks/files/small.txt",
		},
		{
			name:     "example",
			code:     200,
			fileName: "mocks/files/small.txt",
			valid:    true,
		},
	}

	for _, tc := range cases {
		cfg, _ := NewConfig("mocks/config/full.json")

		h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

		// before test we should flush db
		conn := h.App.Redis.Get()
		conn.Do("FLUSHDB")
		conn.Close()

		// remove files
		dirs, err := getDirectories(cfg.Storage.Path)
		if err != nil {
			t.Errorf("Error must be nil but got %v\n", err)
			return
		}

		for _, dir := range dirs {
			d, err := os.Open(path.Join(cfg.Storage.Path, dir))
			if err != nil {
				t.Errorf("Err must be nil but got %v\n", err)
				continue
			}

			items, err := d.Readdir(-1)
			d.Close()

			for _, item := range items {
				os.RemoveAll(path.Join(cfg.Storage.Path, dir, item.Name()))
			}
		}

		if tc.fileName != "" {
			f, err := os.Open(tc.fileName)
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
				continue
			}

			bytesData, err := ioutil.ReadAll(f)
			if err != nil {
				t.Errorf("Error must be nil but got %v\n", err)
				continue
			}
			f.Close()

			if tc.valid {
				hash, err := getSHA256Sum(bytes.NewBuffer(bytesData))
				if err != nil {
					t.Errorf("Error must be nil but got %v\n", err)
					continue
				}

				tc.name = hash + "-" + strconv.FormatInt(time.Now().Unix(), 10) + "-" + strconv.Itoa(rand.Intn(999999))
			}

			h.App.Storage.CreateFile(tc.name, bytes.NewBuffer(bytesData))
			now := time.Now()

			file := FileMeta{
				Hash:      "example",
				Size:      int64(len(bytesData)),
				CreatedAt: &now,
			}

			h.App.Redis.SaveFileMeta(&file)
		}

		//
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "/files/"+tc.name, nil)

		h.ServeHTTP(w, r)

		if w.Code != tc.code {
			t.Errorf("Code must be %d but got %d\n", tc.code, w.Code)
		}

		if tc.code >= 400 {
			errResp := ErrorResponse{}

			json.Unmarshal(w.Body.Bytes(), &errResp)
			if errResp.Error != tc.errMessage {
				t.Errorf("Error message must be %v but got %v\n", tc.errMessage, errResp.Error)
			}
		}
	}
}

func TestHandlerDownloadBandwidth(t *testing.T) {
	cfg, _ := NewConfig("mocks/config/full.json")

	// quick fix
	// update max size for bandwodth test
	cfg.Storage.MaxSize = cfg.RateLimit.Bandwidth.Download + 1024

	h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

	// before test we should flush db
	conn := h.App.Redis.Get()
	conn.Do("FLUSHDB")
	conn.Close()

	strData := strings.Repeat("a", int(cfg.RateLimit.Bandwidth.Download+1))

	h.App.Storage.CreateFile("example", bytes.NewBuffer([]byte(strData)))
	now := time.Now()

	h.App.Redis.SaveFileMeta(&FileMeta{
		Hash:      "example",
		Size:      int64(len(strData)),
		CreatedAt: &now,
	})

	w := httptest.NewRecorder()
	r, _ := http.NewRequest("GET", "/files/example", nil)

	h.ServeHTTP(w, r)

	if w.Code != 403 {
		t.Errorf("Code must be %d but got %d\n", 403, w.Code)
	}

	errResp := ErrorResponse{}

	json.Unmarshal(w.Body.Bytes(), &errResp)
	if errResp.Error != "BYTE_LIMIT_REACHED" {
		t.Errorf("Error message must be %v but got %v\n", "BYTE_LIMIT_REACHED", errResp.Error)
	}
}

func TestHandlerDownloadFileRPS(t *testing.T) {
	cfg, _ := NewConfig("mocks/config/full.json")
	cfg.RateLimit.Bandwidth.Download = cfg.RateLimit.Bandwidth.Download * cfg.RateLimit.Bandwidth.Download
	cfg.RateLimit.RPS.Download = 1

	h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

	// before test we should flush db
	conn := h.App.Redis.Get()
	conn.Do("FLUSHDB")
	conn.Close()

	f, err := os.Open("mocks/files/small.txt")
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}

	bytesData, err := ioutil.ReadAll(f)
	if err != nil {
		t.Errorf("Error must be nil but got %v\n", err)
		return
	}
	f.Close()

	h.App.Storage.CreateFile("example", bytes.NewBuffer(bytesData))
	now := time.Now()

	h.App.Redis.SaveFileMeta(&FileMeta{
		Hash:      "example",
		Size:      int64(len(bytesData)),
		CreatedAt: &now,
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r, _ := http.NewRequest("GET", "/files/example", nil)

		h.ServeHTTP(w, r)

		if i == 1 {
			if w.Code != 429 {
				t.Errorf("Code must be %d but got %d\n", 429, w.Code)
			}

			errResp := ErrorResponse{}

			json.Unmarshal(w.Body.Bytes(), &errResp)
			if errResp.Error != "TOO_MANY_REQUESTS" {
				t.Errorf("Error message must be %v but got %v\n", "TOO_MANY_REQUESTS", errResp.Error)
			}
		}
	}
}

func TestHandler404(t *testing.T) {
	cases := []struct {
		method string
		path   string
		body   []byte
	}{
		{
			method: "GET",
			path:   "/",
		},
		{
			method: "GET",
			path:   "/files",
		},
		{
			method: "GET",
			path:   "/files/file/file",
		},
		{
			method: "DELETE",
			path:   "/files/file/file",
		},
	}

	for _, tc := range cases {
		cfg, _ := NewConfig("mocks/config/full.json")

		h := NewHandler(NewApplication(cfg, NewStorage(cfg.Storage), NewRateLimit(cfg.RateLimit), NewRedis(cfg.Redis)))

		w := httptest.NewRecorder()
		r, _ := http.NewRequest(tc.method, tc.path, bytes.NewBuffer(tc.body))

		h.ServeHTTP(w, r)

		if w.Code != 404 {
			t.Errorf("Code must be %d but got %d\n", 404, w.Code)
		}

		errResp := ErrorResponse{}

		json.Unmarshal(w.Body.Bytes(), &errResp)
		if errResp.Error != "NOT_FOUND" {
			t.Errorf("Error message must be %v but got %v\n", "NOT_FOUND", errResp.Error)
		}
	}
}
