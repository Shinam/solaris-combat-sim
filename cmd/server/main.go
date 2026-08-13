package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/shinam/solaris-simulator/internal/engine"
)

type simRequest struct {
	AB []engine.UnitType `json:"ab"`
	DB []engine.UnitType `json:"db"`
	N  int               `json:"n"`
}

type staticFiles struct {
	dir       string
	indexHTML []byte
	logoPNG   []byte
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8765"
	}

	static, err := loadStaticFiles()
	if err != nil {
		log.Fatalf("static files: %v", err)
	}
	log.Printf("static dir=%s index=%d bytes logo=%d bytes", static.dir, len(static.indexHTML), len(static.logoPNG))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"indexBytes":` + strconv.Itoa(len(static.indexHTML)) + `}`))
	})
	mux.HandleFunc("/simulate", handleSimulate)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/":
			serveBytes(w, static.indexHTML, "text/html; charset=utf-8")
		case "/index.html":
			serveBytes(w, static.indexHTML, "text/html; charset=utf-8")
		case "/logo.png":
			if len(static.logoPNG) == 0 {
				http.NotFound(w, r)
				return
			}
			serveBytes(w, static.logoPNG, "image/png")
		default:
			http.NotFound(w, r)
		}
	})

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Minute,
	}

	log.Printf("solaris combat API listening on http://127.0.0.1:%s", port)
	log.Printf("→ ouvre http://127.0.0.1:%s/ dans le navigateur", port)
	log.Fatal(srv.ListenAndServe())
}

func loadStaticFiles() (*staticFiles, error) {
	dir := os.Getenv("STATIC_DIR")
	if dir == "" {
		for _, candidate := range []string{".", "..", filepath.Join("..", "..")} {
			if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
				dir = candidate
				break
			}
		}
	}
	if dir == "" {
		dir = "."
	}
	dir, _ = filepath.Abs(dir)

	indexPath := filepath.Join(dir, "index.html")
	indexHTML, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}
	if len(indexHTML) < 1000 {
		return nil, fmt.Errorf("index.html too small (%d bytes) at %s", len(indexHTML), indexPath)
	}

	logoPNG, _ := os.ReadFile(filepath.Join(dir, "logo.png"))

	return &staticFiles{dir: dir, indexHTML: indexHTML, logoPNG: logoPNG}, nil
}

func serveBytes(w http.ResponseWriter, data []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	if _, err := w.Write(data); err != nil {
		log.Printf("serveBytes: %v", err)
	}
}

func handleSimulate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8<<20)
	var req simRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.N < 1 {
		req.N = 1000
	}
	if req.N > 10000 {
		http.Error(w, "n max is 10000", http.StatusBadRequest)
		return
	}
	if len(req.AB) == 0 && len(req.DB) == 0 {
		http.Error(w, "ab and db fleets required", http.StatusBadRequest)
		return
	}

	for i := range req.AB {
		if req.AB[i].OrigCount == 0 {
			req.AB[i].OrigCount = req.AB[i].Count
		}
		if req.AB[i].Count == 0 {
			req.AB[i].Count = req.AB[i].OrigCount
		}
	}
	for i := range req.DB {
		if req.DB[i].OrigCount == 0 {
			req.DB[i].OrigCount = req.DB[i].Count
		}
		if req.DB[i].Count == 0 {
			req.DB[i].Count = req.DB[i].OrigCount
		}
	}

	start := time.Now()
	result := engine.RunBatch(req.AB, req.DB, req.N, nil)
	elapsed := time.Since(start)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Sim-Duration-Ms", strconv.FormatInt(elapsed.Milliseconds(), 10))
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("encode error: %v", err)
	}
	log.Printf("simulate n=%d units_atk=%d units_def=%d duration=%s aw=%d dw=%d dr=%d",
		req.N, totalUnits(req.AB), totalUnits(req.DB), elapsed, result.AW, result.DW, result.DR)
}

func totalUnits(fleet []engine.UnitType) int {
	n := 0
	for _, u := range fleet {
		n += u.OrigCount
	}
	return n
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Expose-Headers", "X-Sim-Duration-Ms")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
