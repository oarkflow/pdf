package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/oarkflow/pdf/converter"
)

//go:embed static/*
var staticFS embed.FS

const maxUploadSize = 50 << 20 // 50 MB

// Job represents a conversion job.
type Job struct {
	ID        string
	Filename  string
	Data      []byte
	NumPages  int
	Metadata  map[string]string
	Options   converter.ConvertOptions
	Result    *converter.ConvertResult
	Status    string // "uploaded", "converting", "done", "error"
	Progress  int    // current page being processed
	Total     int    // total pages to convert
	Error     string
	CreatedAt time.Time
}

var (
	jobs   sync.Map
	jobTTL = 30 * time.Minute
)

func main() {
	go cleanupJobs()

	mux := http.NewServeMux()

	// API routes.
	mux.HandleFunc("POST /api/upload", handleUpload)
	mux.HandleFunc("POST /api/convert", handleConvert)
	mux.HandleFunc("GET /api/status/{id}", handleStatus)
	mux.HandleFunc("GET /api/preview/{id}", handlePreview)
	mux.HandleFunc("GET /api/download/{id}", handleDownload)
	mux.HandleFunc("DELETE /api/job/{id}", handleDelete)
	mux.HandleFunc("POST /api/batch", handleBatch)
	mux.HandleFunc("GET /api/metadata/{id}", handleMetadata)

	// Serve static files (SPA).
	mux.HandleFunc("GET /", handleStatic)

	addr := ":8080"
	log.Printf("PDF-to-HTML converter running at http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/static/index.html"
	} else {
		path = "/static" + path
	}

	data, err := staticFS.ReadFile(path[1:]) // strip leading /
	if err != nil {
		// Try index.html for SPA routing.
		data, err = staticFS.ReadFile("static/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	switch {
	case len(path) > 5 && path[len(path)-5:] == ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case len(path) > 4 && path[len(path)-4:] == ".css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case len(path) > 3 && path[len(path)-3:] == ".js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}

	w.Write(data)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		jsonError(w, "File too large (max 50MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		jsonError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		jsonError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Try to open the PDF to validate it and get page count.
	password := r.FormValue("password")
	opts := converter.ConvertOptions{Password: password}
	conv, err := converter.New(data, opts)
	if err != nil {
		jsonError(w, fmt.Sprintf("Invalid PDF: %v", err), http.StatusBadRequest)
		return
	}

	id := generateID()
	job := &Job{
		ID:        id,
		Filename:  header.Filename,
		Data:      data,
		NumPages:  conv.NumPages(),
		Metadata:  conv.Metadata(),
		Status:    "uploaded",
		CreatedAt: time.Now(),
	}
	jobs.Store(id, job)

	jsonResponse(w, map[string]interface{}{
		"id":       id,
		"filename": header.Filename,
		"pages":    conv.NumPages(),
		"metadata": conv.Metadata(),
	})
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID            string `json:"id"`
		Pages         []int  `json:"pages"`
		Mode          string `json:"mode"`
		ExtractImages bool   `json:"extractImages"`
		DetectTables  bool   `json:"detectTables"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	val, ok := jobs.Load(req.ID)
	if !ok {
		jsonError(w, "Job not found", http.StatusNotFound)
		return
	}
	job := val.(*Job)

	if job.Status == "converting" {
		jsonError(w, "Conversion already in progress", http.StatusConflict)
		return
	}

	job.Options = converter.ConvertOptions{
		Pages:         req.Pages,
		Mode:          req.Mode,
		ExtractImages: req.ExtractImages,
		DetectTables:  req.DetectTables,
	}

	job.Status = "converting"
	job.Progress = 0
	total := job.NumPages
	if len(req.Pages) > 0 {
		total = len(req.Pages)
	}
	job.Total = total

	// Run conversion in background.
	go func() {
		conv, err := converter.New(job.Data, job.Options)
		if err != nil {
			job.Status = "error"
			job.Error = err.Error()
			return
		}

		conv.SetProgressCallback(func(page, total int) {
			job.Progress = page + 1
			job.Total = total
		})

		result, err := conv.Convert()
		if err != nil {
			job.Status = "error"
			job.Error = err.Error()
			return
		}

		job.Result = result
		job.Status = "done"
	}()

	jsonResponse(w, map[string]interface{}{
		"status": "converting",
		"total":  total,
	})
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	val, ok := jobs.Load(id)
	if !ok {
		jsonError(w, "Job not found", http.StatusNotFound)
		return
	}
	job := val.(*Job)

	resp := map[string]interface{}{
		"status":   job.Status,
		"progress": job.Progress,
		"total":    job.Total,
	}
	if job.Error != "" {
		resp["error"] = job.Error
	}
	jsonResponse(w, resp)
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	val, ok := jobs.Load(id)
	if !ok {
		jsonError(w, "Job not found", http.StatusNotFound)
		return
	}
	job := val.(*Job)

	if job.Status != "done" || job.Result == nil {
		jsonError(w, "Conversion not complete", http.StatusPreconditionFailed)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(job.Result.HTML))
}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	val, ok := jobs.Load(id)
	if !ok {
		jsonError(w, "Job not found", http.StatusNotFound)
		return
	}
	job := val.(*Job)

	if job.Status != "done" || job.Result == nil {
		jsonError(w, "Conversion not complete", http.StatusPreconditionFailed)
		return
	}

	filename := job.Filename
	if len(filename) > 4 && filename[len(filename)-4:] == ".pdf" {
		filename = filename[:len(filename)-4]
	}
	filename += ".html"

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Write([]byte(job.Result.HTML))
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	jobs.Delete(id)
	jsonResponse(w, map[string]interface{}{"deleted": true})
}

func handleBatch(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize*10)

	if err := r.ParseMultipartForm(maxUploadSize * 10); err != nil {
		jsonError(w, "Files too large", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		jsonError(w, "No files provided", http.StatusBadRequest)
		return
	}

	var results []map[string]interface{}
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			results = append(results, map[string]interface{}{
				"filename": fileHeader.Filename,
				"error":    "Failed to open file",
			})
			continue
		}

		data, err := io.ReadAll(file)
		file.Close()
		if err != nil {
			results = append(results, map[string]interface{}{
				"filename": fileHeader.Filename,
				"error":    "Failed to read file",
			})
			continue
		}

		conv, err := converter.New(data, converter.ConvertOptions{})
		if err != nil {
			results = append(results, map[string]interface{}{
				"filename": fileHeader.Filename,
				"error":    fmt.Sprintf("Invalid PDF: %v", err),
			})
			continue
		}

		id := generateID()
		job := &Job{
			ID:        id,
			Filename:  fileHeader.Filename,
			Data:      data,
			NumPages:  conv.NumPages(),
			Metadata:  conv.Metadata(),
			Status:    "uploaded",
			CreatedAt: time.Now(),
		}
		jobs.Store(id, job)

		results = append(results, map[string]interface{}{
			"id":       id,
			"filename": fileHeader.Filename,
			"pages":    conv.NumPages(),
		})
	}

	jsonResponse(w, map[string]interface{}{"files": results})
}

func handleMetadata(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	val, ok := jobs.Load(id)
	if !ok {
		jsonError(w, "Job not found", http.StatusNotFound)
		return
	}
	job := val.(*Job)

	jsonResponse(w, map[string]interface{}{
		"filename": job.Filename,
		"pages":    job.NumPages,
		"metadata": job.Metadata,
	})
}

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 12)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func cleanupJobs() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		now := time.Now()
		jobs.Range(func(key, value interface{}) bool {
			job := value.(*Job)
			if now.Sub(job.CreatedAt) > jobTTL {
				jobs.Delete(key)
			}
			return true
		})
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
