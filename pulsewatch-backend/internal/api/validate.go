package api

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"pulsewatch-backend/internal/netguard"
)

const (
	maxNameLength = 200
	maxURLLength  = 2048
)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// requireValidID checks a path param looks like a UUID before it ever
// reaches a query. Without this, a malformed ID (a stray link, a typo, a
// bot probing the API) makes Postgres reject the WHERE clause with
// "invalid input syntax for uuid", an error type the handlers don't
// special-case, so it falls through to a generic 500 instead of a 404.
// Reports the same "not found" response either way, so callers can't
// distinguish "malformed" from "doesn't exist".
func requireValidID(w http.ResponseWriter, id string) bool {
	if !uuidPattern.MatchString(id) {
		writeError(w, http.StatusNotFound, "monitor not found")
		return false
	}
	return true
}

var allowedMonitorMethods = map[string]bool{"GET": true, "HEAD": true, "POST": true}

func validateName(name string) error {
	if len(name) > maxNameLength {
		return fmt.Errorf("name must be %d characters or fewer", maxNameLength)
	}
	return nil
}

// validateHTTPURL is shared by monitor URLs and the webhook URL; both need
// a real http(s) target, not just a non-empty string.
//
// This only catches the obvious case: a literal loopback/private/link-local
// IP or "localhost" typed directly into the field. It's a cheap, immediate
// rejection so a request doesn't just silently create a monitor that will
// always report down with a confusing error. It is NOT the real defense
// against SSRF, since a hostname can resolve to a different address later,
// or an attacker's own domain can simply resolve straight to an internal
// IP without ever mentioning it in the URL string. The checker's dial-time
// guard (internal/netguard) is what actually enforces this on every
// request; this is just a fast-fail for the common case.
func validateHTTPURL(raw string) error {
	if len(raw) > maxURLLength {
		return fmt.Errorf("url must be %d characters or fewer", maxURLLength)
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("url is not valid: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("url must start with http:// or https://")
	}
	if u.Host == "" {
		return fmt.Errorf("url must include a host")
	}
	hostname := u.Hostname()
	if strings.EqualFold(hostname, "localhost") {
		return fmt.Errorf("url must not point to a local or internal address")
	}
	if ip := net.ParseIP(hostname); ip != nil && !netguard.IsPublicIP(ip) {
		return fmt.Errorf("url must not point to a local or internal address")
	}
	return nil
}

func validateStatusCode(code int) error {
	if code < 100 || code > 599 {
		return fmt.Errorf("expected_status_code must be between 100 and 599")
	}
	return nil
}

// validateMethod normalizes and restricts to methods that make sense for a
// health check; the frontend only ever offers these three.
func validateMethod(raw string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if !allowedMonitorMethods[normalized] {
		return "", fmt.Errorf("method must be one of GET, HEAD, POST")
	}
	return normalized, nil
}
