package middleware

import (
	"net"

	"github.com/labstack/echo/v5"
)

// cgnatRange is the RFC 6598 shared address space (100.64.0.0/10). Hops in it
// are treated as trusted proxies on top of Echo's defaults (loopback,
// link-local, private), matching axumbackend's client_identity and
// fiberbackend's TrustProxyConfig.
var cgnatRange = &net.IPNet{IP: net.IPv4(100, 64, 0, 0).To4(), Mask: net.CIDRMask(10, 32)}

// IPExtractor returns the strategy c.RealIP() uses to resolve the client IP.
//
// With trustProxy, X-Forwarded-For is walked right-to-left (the peer address
// included) and the first hop outside the trusted ranges (loopback,
// link-local, private, CGNAT) is returned. A client cannot spoof its IP by
// injecting X-Forwarded-For unless the request actually arrives through a
// trusted proxy hop. When every hop is trusted the leftmost one is returned.
// Only enable this behind a reverse proxy that sets the header.
//
// Without trustProxy, the peer address is used and proxy headers are ignored.
func IPExtractor(trustProxy bool) echo.IPExtractor {
	if trustProxy {
		return echo.ExtractIPFromXFFHeader(echo.TrustIPRange(cgnatRange))
	}
	return echo.ExtractIPDirect()
}
