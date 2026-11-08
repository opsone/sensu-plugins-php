package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sensu/sensu-plugin-sdk/sensu"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		body     string
		expected string
		want     Result
	}{
		{"pong", 200, "pong", "pong", Result{sensu.CheckStateOK, "pong"}},
		{"trailing newline", 200, "pong\n", "pong", Result{sensu.CheckStateOK, "pong"}},
		{"custom response", 200, "alive", "alive", Result{sensu.CheckStateOK, "alive"}},
		{"wrong response", 200, "File not found.", "pong", Result{sensu.CheckStateCritical, "File not found. instead of pong"}},
		{"empty body", 200, "", "pong", Result{sensu.CheckStateCritical, " instead of pong"}},
		{"long body truncated", 200, strings.Repeat("x", 150), "pong", Result{sensu.CheckStateCritical, strings.Repeat("x", 100) + "… instead of pong"}},
		{"not found", 404, "pong", "pong", Result{sensu.CheckStateCritical, "Error, http response code: 404"}},
		{"bad gateway", 502, "", "pong", Result{sensu.CheckStateCritical, "Error, http response code: 502"}},
		{"redirect", 301, "", "pong", Result{sensu.CheckStateCritical, "Error, http response code: 301"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluate(tt.code, tt.body, tt.expected); got != tt.want {
				t.Errorf("evaluate() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestPingURL(t *testing.T) {
	tests := []struct {
		host, path string
		port       int
		want       string
	}{
		{"127.0.0.1", "fpm-ping", 80, "http://127.0.0.1:80/fpm-ping"},
		{"localhost", "/fpm-ping", 8080, "http://localhost:8080/fpm-ping"},
		{"::1", "status/ping", 80, "http://[::1]:80/status/ping"},
	}
	for _, tt := range tests {
		if got := pingURL(tt.host, tt.port, tt.path); got != tt.want {
			t.Errorf("pingURL(%q, %d, %q) = %q, want %q", tt.host, tt.port, tt.path, got, tt.want)
		}
	}
}

func TestFetch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/fpm-ping":
			w.Write([]byte("pong"))
		case "/redirect":
			http.Redirect(w, r, "/fpm-ping", http.StatusMovedPermanently)
		case "/slow":
			time.Sleep(500 * time.Millisecond)
			w.Write([]byte("pong"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	tests := []struct {
		path      string
		wantState int
	}{
		{"/fpm-ping", sensu.CheckStateOK},
		{"/redirect", sensu.CheckStateCritical},
		{"/nope", sensu.CheckStateCritical},
		{"/slow", sensu.CheckStateCritical},
	}
	for _, tt := range tests {
		got := fetch(srv.URL+tt.path, 200*time.Millisecond, "pong")
		if got.State != tt.wantState {
			t.Errorf("fetch(%s) = %+v, want state %d", tt.path, got, tt.wantState)
		}
	}

	srv.Close()
	if got := fetch(srv.URL+"/fpm-ping", time.Second, "pong"); got.State != sensu.CheckStateCritical {
		t.Errorf("fetch on closed server = %+v, want critical", got)
	}
}
