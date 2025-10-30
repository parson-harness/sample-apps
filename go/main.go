package main

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Testing Embed everything under static/
//
//go:embed static/*
var embeddedFS embed.FS

// Also embed index.html specifically for easy serving at "/"
//
//go:embed static/index.html
var indexHTML []byte

type AppInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Commit      string `json:"commit"` // NEW
	Environment string `json:"environment"`
	BuildTime   string `json:"buildTime"`
	Uptime      string `json:"uptime"`
	Hostname    string `json:"hostname"`
}

var (
	startTime  = time.Now()
	version    = getenv("APP_VERSION", "1.0.0") // overridden by -ldflags main.version
	env        = getenv("APP_ENV", "development")
	buildTime  = os.Getenv("BUILD_TIME") // overridden by -ldflags main.buildTime
	commit     = "unknown"               // overridden by -ldflags main.commit  // NEW
	readyAfter = 2 * time.Second
	logger     = slog.New(slog.NewJSONHandler(os.Stdout, nil))
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	port := getenv("PORT", "8080")

	// Serve /static/* from the embedded filesystem (rooted at "static")
	sub, err := fsSub("static")
	if err != nil {
		log.Fatalf("failed to sub FS: %v", err)
	}
	staticHandler := http.FileServer(http.FS(sub))

	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", staticHandler))

	mux.Handle("/", chain(http.HandlerFunc(homeHandler), withSecurityHeaders(), withLogging()))
	mux.Handle("/api/info", chain(http.HandlerFunc(infoHandler), withSecurityHeaders(), withLogging()))
	mux.Handle("/version", chain(http.HandlerFunc(versionHandler), withSecurityHeaders(), withLogging()))

	// Existing probe paths
	mux.Handle("/health", chain(http.HandlerFunc(healthHandler), withSecurityHeaders(), withLogging()))
	mux.Handle("/live", chain(http.HandlerFunc(liveHandler), withSecurityHeaders(), withLogging()))
	mux.Handle("/ready", chain(http.HandlerFunc(readyHandler), withSecurityHeaders(), withLogging()))

	// Kube-style aliases (no change to handlers)
	mux.Handle("/healthz", chain(http.HandlerFunc(healthHandler), withSecurityHeaders(), withLogging()))
	mux.Handle("/livez", chain(http.HandlerFunc(liveHandler), withSecurityHeaders(), withLogging()))
	mux.Handle("/readyz", chain(http.HandlerFunc(readyHandler), withSecurityHeaders(), withLogging()))

	mux.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second, // NEW
		WriteTimeout:      10 * time.Second, // NEW
		IdleTimeout:       60 * time.Second, // NEW
	}

	logger.Info("server starting", "port", port, "commit", commit, "version", version, "env", env, "buildTime", buildTime)

	// start server
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	logger.Info("shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server shutdown error", "err", err)
	} else {
		logger.Info("server stopped cleanly")
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.Copy(w, bytesReader(indexHTML))
}
func versionHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	info := AppInfo{
		Name:        "Harness Demo App",
		Version:     version,
		Commit:      commit,
		Environment: env,
		BuildTime:   buildTime,
		Uptime:      time.Since(startTime).Truncate(time.Second).String(),
		Hostname:    hostname,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

func infoHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	info := AppInfo{
		Name:        "Harness Demo App",
		Version:     version,
		Environment: env,
		BuildTime:   buildTime,
		Uptime:      time.Since(startTime).Truncate(time.Second).String(),
		Hostname:    hostname,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, `{"status":"healthy"}`)
}

func liveHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, `{"status":"alive"}`)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	if time.Since(startTime) < readyAfter {
		writeJSON(w, http.StatusServiceUnavailable, `{"status":"warming"}`)
		return
	}
	writeJSON(w, http.StatusOK, `{"status":"ready"}`)
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprint(w, body)
}

// --- helpers / middleware ---

// fsSub returns an fs.FS rooted at subdir (e.g., "static") from embeddedFS
func fsSub(dir string) (fs.FS, error) {
	return fs.Sub(embeddedFS, dir)
}

func bytesReader(b []byte) io.Reader { return bytes.NewReader(b) }

func withSecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; script-src 'self'")
			next.ServeHTTP(w, r)
		})
	}
}

func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type rwCapture struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *rwCapture) WriteHeader(code int) { w.status = code; w.ResponseWriter.WriteHeader(code) }
func (w *rwCapture) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func withLogging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			cw := &rwCapture{ResponseWriter: w}
			next.ServeHTTP(cw, r)
			slog.Default().Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", cw.status,
				"bytes", cw.size,
				"remote", r.RemoteAddr,
				"dur_ms", time.Since(start).Milliseconds(),
			)
		})
	}
}
