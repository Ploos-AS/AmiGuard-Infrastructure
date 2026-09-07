package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "time"
)

type health struct {
    Status string `json:"status"`
    Service string `json:"service"`
}

func main() {
    addr := env("AMIGUARD_SUBMIT_ADDR", "127.0.0.1:8080")

    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(health{Status: "ok", Service: "amiguard-submit"})
    })
    mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }
        w.Header().Set("Content-Type", "text/html; charset=utf-8")
        w.Header().Set("Cache-Control", "no-store")
        fmt.Fprint(w, `<!doctype html><html><head><meta charset="utf-8"><title>AmiGuard Sample Submission</title></head><body><main><h1>AmiGuard Sample Submission</h1><p>The secure submission channel is being prepared.</p><p>Uploads are disabled until the infrastructure hardening gate has passed.</p></main></body></html>`)
    })

    srv := &http.Server{
        Addr: addr,
        Handler: mux,
        ReadHeaderTimeout: 5 * time.Second,
        ReadTimeout: 10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout: 30 * time.Second,
        MaxHeaderBytes: 16 << 10,
    }

    log.Printf("amiguard-submit listening on %s (uploads disabled)", addr)
    log.Fatal(srv.ListenAndServe())
}

func env(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}
