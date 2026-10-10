// Command e2e_autostart_attach is the fast architecture unit race (mock proxy).
//
// For the real-machine multi-IDE proof that drives the trim binary, run:
//
//	python scripts/e2e_autostart_attach_live.py
//
// This harness still covers concurrent health-then-attach + second-bind refuse +
// shutdown observe-down without requiring cli/.env or site_messages.
//
// Run from this directory:
//
//	go run .
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const ideClients = 4 // Cursor + VS Code + JetBrains-class + daemon sidecar

func main() {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fail("listen: %v", err)
	}
	addr := ln.Addr().String()
	healthURL := "http://" + addr + "/health"
	shutdownURL := "http://" + addr + "/v1/control/shutdown"

	var healthy atomic.Bool
	healthy.Store(true)
	var shutdownHits atomic.Int64

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok", "proxy": "trim"})
	})
	mux.HandleFunc("/v1/control/shutdown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		shutdownHits.Add(1)
		healthy.Store(false)
		w.WriteHeader(http.StatusNoContent)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 2 * time.Second}
	go func() { _ = srv.Serve(ln) }()
	defer func() { _ = srv.Close() }()

	// Primary (IDE #1 / first start) must be healthy before attachers race in.
	if !waitHealth(healthURL, true, 2*time.Second) {
		fail("primary health never became ok")
	}
	fmt.Println("OK primary healthy:", healthURL)

	// Multi-IDE attach race: all clients must see healthy and refuse a second bind.
	var wg sync.WaitGroup
	var attachFails atomic.Int64
	var bindAttempts atomic.Int64
	var bindSucceeded atomic.Int64
	for i := 0; i < ideClients; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			// Health-then-attach (same order as CLI start / daemon / extension).
			if !proxyHealthOK(healthURL, 500*time.Millisecond) {
				attachFails.Add(1)
				fmt.Fprintf(os.Stderr, "FAIL client %d attach health\n", id)
				return
			}
			// Prove double-bind is refused while primary owns the port.
			bindAttempts.Add(1)
			second, err := net.Listen("tcp", addr)
			if err == nil {
				bindSucceeded.Add(1)
				_ = second.Close()
				fmt.Fprintf(os.Stderr, "FAIL client %d double-bound %s\n", id, addr)
				return
			}
		}(i + 1)
	}
	wg.Wait()
	if attachFails.Load() != 0 {
		fail("%d/%d clients failed attach health", attachFails.Load(), ideClients)
	}
	if bindSucceeded.Load() != 0 {
		fail("%d clients double-bound the primary port", bindSucceeded.Load())
	}
	if bindAttempts.Load() != ideClients {
		fail("expected %d bind attempts, got %d", ideClients, bindAttempts.Load())
	}
	fmt.Printf("OK %d clients attached concurrently; all second-bind attempts refused\n", ideClients)

	// Pref-off: one enforcer posts shutdown (dashboard uncheck / managed-off path).
	req, err := http.NewRequest(http.MethodPost, shutdownURL, nil)
	if err != nil {
		fail("shutdown req: %v", err)
	}
	client := &http.Client{Timeout: 500 * time.Millisecond}
	res, err := client.Do(req)
	if err != nil {
		fail("shutdown: %v", err)
	}
	_ = res.Body.Close()
	if shutdownHits.Load() < 1 {
		fail("shutdown not hit")
	}
	if !waitHealth(healthURL, false, 2*time.Second) {
		fail("proxy still healthy after shutdown")
	}

	// All IDEs re-check and observe down (stop enforcing / no thrash restart).
	var downFails atomic.Int64
	wg = sync.WaitGroup{}
	for i := 0; i < ideClients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if proxyHealthOK(healthURL, 300*time.Millisecond) {
				downFails.Add(1)
			}
		}()
	}
	wg.Wait()
	if downFails.Load() != 0 {
		fail("%d clients still saw healthy after pref-off", downFails.Load())
	}
	fmt.Println("OK enforcer shutdown stopped proxy; all clients observe down (pref-off path)")
	fmt.Println("PASS e2e_autostart_attach")
}

func proxyHealthOK(url string, timeout time.Duration) bool {
	client := &http.Client{Timeout: timeout}
	res, err := client.Get(url)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return false
	}
	var body struct {
		Status string `json:"status"`
		Proxy  string `json:"proxy"`
	}
	if json.NewDecoder(res.Body).Decode(&body) != nil {
		return false
	}
	return body.Status == "ok" && body.Proxy == "trim"
}

func waitHealth(url string, wantOK bool, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		ok := proxyHealthOK(url, 200*time.Millisecond)
		if ok == wantOK {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "FAIL "+format+"\n", args...)
	os.Exit(1)
}
