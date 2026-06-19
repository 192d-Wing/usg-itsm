// Package gw holds the gateway's internal routing/BFF logic.
package gw

import (
	"crypto/tls"
	"net/url"
	"strings"
	"time"

	"github.com/192d-Wing/usg-itsm/pkg/httpx"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/proxy"
	"github.com/valyala/fasthttp"
)

// DefaultUpstreamTimeout bounds a single proxied request to an upstream.
const DefaultUpstreamTimeout = 30 * time.Second

// NewUpstreamClient builds a fasthttp client for service-to-service calls with
// the given TLS 1.3 config (ADR-0007).
func NewUpstreamClient(tlsCfg *tls.Config) *fasthttp.Client {
	return &fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		DisablePathNormalizing:   true,
		TLSConfig:                tlsCfg,
	}
}

// Proxy returns a handler that forwards the request to upstream, preserving the
// method, path, query, body, and headers (including Authorization, so the
// upstream re-validates the token — defense in depth). A failed upstream call
// becomes 502 Bad Gateway.
func Proxy(upstream string, client *fasthttp.Client, timeout time.Duration) fiber.Handler {
	base := strings.TrimRight(upstream, "/")
	if timeout <= 0 {
		timeout = DefaultUpstreamTimeout
	}
	// Derive the upstream host so the forwarded Host header matches the cert /
	// SNI in verify mode (rather than leaking the gateway's inbound Host).
	var host string
	if u, err := url.Parse(upstream); err == nil {
		host = u.Host
	}
	return func(c *fiber.Ctx) error {
		if host != "" {
			c.Request().Header.SetHost(host)
		}
		// fiber's proxy.Do replaces the request URI entirely, so append the
		// original path+query to the upstream base.
		if err := proxy.DoTimeout(c, base+c.OriginalURL(), timeout, client); err != nil {
			return httpx.WriteError(c, fiber.StatusBadGateway, "bad_gateway", "upstream unavailable")
		}
		return nil
	}
}
