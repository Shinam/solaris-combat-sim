package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/shinam/solaris-simulator/internal/engine"
)

type simRequest struct {
	AB []engine.UnitType `json:"ab"`
	DB []engine.UnitType `json:"db"`
	N  int               `json:"n"`
}

type progressMsg struct {
	Type  string `json:"type"`
	Done  int    `json:"done"`
	Total int    `json:"total"`
	AW    int    `json:"aw"`
	DW    int    `json:"dw"`
	DR    int    `json:"dr"`
	Ms    int64  `json:"ms"`
}

type staticStore struct {
	dir string
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9876"
	}

	static, err := newStaticStore()
	if err != nil {
		log.Fatalf("static files: %v", err)
	}
	index, _ := static.readIndex()
	log.Printf("static dir=%s index=%d bytes (rechargé à chaque requête)", static.dir, len(index))

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		b, err := static.readIndex()
		n := 0
		if err == nil {
			n = len(b)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(`{"ok":true,"indexBytes":` + strconv.Itoa(n) + `}`))
	})
	mux.HandleFunc("/simulate", handleSimulate)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/", "/index.html":
			b, err := static.readIndex()
			if err != nil {
				http.Error(w, "index.html: "+err.Error(), http.StatusInternalServerError)
				return
			}
			serveBytes(w, b, "text/html; charset=utf-8")
		case "/logo.png":
			b, err := static.readLogo()
			if err != nil || len(b) == 0 {
				http.NotFound(w, r)
				return
			}
			serveBytes(w, b, "image/png")
		default:
			http.NotFound(w, r)
		}
	})

	addr := ":" + port
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("port %s déjà pris — tue l'ancien process :\n  kill -9 $(lsof -tiTCP:%s -sTCP:LISTEN)\nerreur: %v", port, port, err)
	}

	srv := &http.Server{
		Handler:           withCORS(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      15 * time.Minute,
	}

	log.Printf("solaris combat API listening on http://127.0.0.1:%s", port)
	log.Printf("→ ouvre http://127.0.0.1:%s/  (hard refresh Cmd+Shift+R)", port)
	log.Fatal(srv.Serve(ln))
}

func newStaticStore() (*staticStore, error) {
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
	if _, err := os.Stat(filepath.Join(dir, "index.html")); err != nil {
		return nil, fmt.Errorf("index.html introuvable dans %s", dir)
	}
	return &staticStore{dir: dir}, nil
}

func (s *staticStore) readIndex() ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(s.dir, "index.html"))
	if err != nil {
		return nil, err
	}
	if len(b) < 1000 {
		return nil, fmt.Errorf("index.html trop petit (%d bytes)", len(b))
	}
	return b, nil
}

func (s *staticStore) readLogo() ([]byte, error) {
	return os.ReadFile(filepath.Join(s.dir, "logo.png"))
}

func serveBytes(w http.ResponseWriter, data []byte, contentType string) {
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "no-store, max-age=0")
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

	wantStream := r.URL.Query().Get("stream") != "0"
	accept := r.Header.Get("Accept")
	if accept == "application/json" {
		wantStream = false
	}

	start := time.Now()
	ua, ud := totalUnits(req.AB), totalUnits(req.DB)
	log.Printf("simulate start n=%d units_atk=%d units_def=%d stream=%v", req.N, ua, ud, wantStream)

	if !wantStream {
		result := engine.RunBatch(req.AB, req.DB, req.N, nil)
		elapsed := time.Since(start)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Sim-Duration-Ms", strconv.FormatInt(elapsed.Milliseconds(), 10))
		if err := json.NewEncoder(w).Encode(result); err != nil {
			log.Printf("encode error: %v", err)
		}
		log.Printf("simulate done n=%d duration=%s aw=%d dw=%d dr=%d", req.N, elapsed, result.AW, result.DW, result.DR)
		return
	}

	w.Header().Set("Content-Type", "application/x-ndjson")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	var writeMu sync.Mutex
	enc := json.NewEncoder(w)
	writeLine := func(v any) error {
		writeMu.Lock()
		defer writeMu.Unlock()
		if err := enc.Encode(v); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	_ = writeLine(progressMsg{Type: "progress", Done: 0, Total: req.N, Ms: 0})

	result := engine.RunBatch(req.AB, req.DB, req.N, func(done, total, aw, dw, dr int) {
		_ = writeLine(progressMsg{
			Type:  "progress",
			Done:  done,
			Total: total,
			AW:    aw,
			DW:    dw,
			DR:    dr,
			Ms:    time.Since(start).Milliseconds(),
		})
	})
	elapsed := time.Since(start)
	result.Type = "done"
	result.DurationMs = elapsed.Milliseconds()
	if err := writeLine(result); err != nil {
		log.Printf("stream encode error: %v", err)
		return
	}
	log.Printf("simulate done n=%d units_atk=%d units_def=%d duration=%s aw=%d dw=%d dr=%d",
		req.N, ua, ud, elapsed, result.AW, result.DW, result.DR)
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
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.Header().Set("Access-Control-Expose-Headers", "X-Sim-Duration-Ms")
		w.Header().Set("Vary", "Origin")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
