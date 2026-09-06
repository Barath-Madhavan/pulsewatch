// Package netguard blocks outbound requests to non-public network
// addresses. PulseWatch's checker makes real server-side HTTP requests to
// whatever URL a visitor enters, which on a publicly hosted instance is a
// classic SSRF vector: without this, someone could point a monitor at the
// server's own internal network, or at a cloud provider's metadata
// endpoint (169.254.169.254), and use the check results (up/down, timing,
// status code) to probe or exfiltrate from infrastructure they were never
// meant to reach.
package netguard

import (
	"fmt"
	"net"
	"syscall"
)

// IsPublicIP reports whether ip is safe for the server to connect to on a
// visitor's behalf: not loopback, not a private or carrier-grade-NAT
// range, not link-local (this also covers the cloud metadata address,
// which lives in the link-local block), and not multicast or unspecified.
func IsPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() {
		return false
	}
	// 100.64.0.0/10 (carrier-grade NAT) isn't covered by net.IP.IsPrivate,
	// which only implements RFC 1918 / RFC 4193.
	if ip4 := ip.To4(); ip4 != nil && ip4[0] == 100 && ip4[1]&0xc0 == 64 {
		return false
	}
	return true
}

// DialControl is the actual enforcement point: pass it as net.Dialer.Control
// on any http.Transport that reaches a user-supplied URL. It runs after DNS
// resolution but before the socket connects, checking the real destination
// IP on every single dial. That matters because a one-time check against
// the hostname when a monitor is created can't catch a DNS record that
// resolves to a public address today and a private one tomorrow (or an
// attacker-controlled domain that was always going to resolve straight to
// an internal IP); this closes that gap by re-verifying on every check.
func DialControl(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("invalid dial address %q: %w", address, err)
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return fmt.Errorf("could not parse resolved IP from %q", address)
	}
	if !IsPublicIP(ip) {
		return fmt.Errorf("refusing to connect to non-public address %s", ip)
	}
	return nil
}
