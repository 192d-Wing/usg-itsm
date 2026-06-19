package gw_test

import (
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/tlsconf"
	"github.com/192d-Wing/usg-itsm/services/gateway/internal/gw"
	"github.com/gofiber/fiber/v2"
)

type echo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Query  string `json:"query"`
	Auth   string `json:"auth"`
	Body   string `json:"body"`
}

// startBackend spins up a TLS 1.3 backend that echoes the request, returning its
// base URL.
func startBackend(t *testing.T) string {
	t.Helper()
	cert, err := tlsconf.SelfSigned("localhost")
	if err != nil {
		t.Fatalf("self-signed: %v", err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", tlsconf.Server(cert))
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	handler := func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusCreated).JSON(echo{
			Method: c.Method(),
			Path:   c.Path(),
			Query:  string(c.Request().URI().QueryString()),
			Auth:   c.Get(fiber.HeaderAuthorization),
			Body:   string(c.Body()),
		})
	}
	app.All("/api/v1/tickets", handler)
	app.All("/api/v1/tickets/*", handler)

	go func() { _ = app.Listener(ln) }()
	t.Cleanup(func() { _ = app.Shutdown() })

	addr := ln.Addr().String()
	waitReachable(t, addr)
	return "https://" + addr
}

func waitReachable(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("backend %s not reachable", addr)
}

func frontApp(upstream string) *fiber.App {
	clientTLS, _ := tlsconf.Client("", true) // dev skip-verify for self-signed
	client := gw.NewUpstreamClient(clientTLS)
	h := gw.Proxy(upstream, client, 5*time.Second)
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.All("/api/v1/tickets", h)
	app.All("/api/v1/tickets/*", h)
	return app
}

func TestProxy_ForwardsRequest(t *testing.T) {
	upstream := startBackend(t)
	app := frontApp(upstream)

	req := httptest.NewRequest(fiber.MethodPost, "/api/v1/tickets/abc?foo=bar", strings.NewReader(`{"x":1}`))
	req.Header.Set(fiber.HeaderAuthorization, "Bearer tok123")
	req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if resp.StatusCode != fiber.StatusCreated {
		t.Fatalf("status passthrough: want 201, got %d", resp.StatusCode)
	}

	var got echo
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode echo: %v (%s)", err, body)
	}
	if got.Method != fiber.MethodPost {
		t.Errorf("method = %q", got.Method)
	}
	if got.Path != "/api/v1/tickets/abc" {
		t.Errorf("path = %q", got.Path)
	}
	if got.Query != "foo=bar" {
		t.Errorf("query = %q", got.Query)
	}
	if got.Auth != "Bearer tok123" {
		t.Errorf("auth not forwarded: %q", got.Auth)
	}
	if got.Body != `{"x":1}` {
		t.Errorf("body = %q", got.Body)
	}
}

func TestProxy_UpstreamDownIs502(t *testing.T) {
	// Nothing listening on this port.
	app := frontApp("https://127.0.0.1:1")
	req := httptest.NewRequest(fiber.MethodGet, "/api/v1/tickets", nil)
	resp, err := app.Test(req, 5000)
	if err != nil {
		t.Fatalf("test: %v", err)
	}
	if resp.StatusCode != fiber.StatusBadGateway {
		t.Fatalf("want 502, got %d", resp.StatusCode)
	}
}
