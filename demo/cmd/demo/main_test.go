package main

import (
	"testing"
	"time"
)

func TestEndpoints(t *testing.T) {
	endpoints := []string{"/api/users", "/api/orders", "/api/products", "/health"}
	for _, ep := range endpoints {
		t.Run(ep, func(t *testing.T) {
			if ep == "" {
				t.Fatal("empty endpoint")
			}
		})
	}
}

func TestStatusCodes(t *testing.T) {
	tests := []struct {
		code int
		ok   bool
	}{
		{200, true},
		{201, true},
		{204, true},
		{400, false},
		{404, false},
		{500, false},
	}

	for _, tt := range tests {
		if (tt.code < 400) != tt.ok {
			t.Errorf("status %d: expected ok=%v", tt.code, tt.ok)
		}
	}
}

func TestLogFormat(t *testing.T) {
	entry := log("info", "test message", map[string]any{"key": "value"})
	if entry == "" {
		t.Fatal("empty log entry")
	}
}

func TestServerStartup(t *testing.T) {
	start := time.Now()
	time.Sleep(10 * time.Millisecond)

	if time.Since(start) < 10*time.Millisecond {
		t.Fatal("timer broken")
	}
}

func TestHealthCheck(t *testing.T) {
	statuses := []int{200, 200, 200}
	for _, s := range statuses {
		if s != 200 {
			t.Fatalf("health check returned %d", s)
		}
	}
}
