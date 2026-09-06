package netguard

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPublicIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"ordinary public IPv4", "8.8.8.8", true},
		{"ordinary public IPv6", "2001:4860:4860::8888", true},
		{"loopback IPv4", "127.0.0.1", false},
		{"loopback IPv6", "::1", false},
		{"private 10.0.0.0/8", "10.0.0.5", false},
		{"private 172.16.0.0/12", "172.16.5.1", false},
		{"private 192.168.0.0/16", "192.168.1.1", false},
		{"unique local IPv6 (fc00::/7)", "fd00::1", false},
		{"carrier-grade NAT 100.64.0.0/10", "100.64.0.1", false},
		{"just outside CGNAT range, still public", "100.63.255.255", true},
		{"just outside CGNAT range on the other side, still public", "100.128.0.0", true},
		{"link-local unicast incl. cloud metadata", "169.254.169.254", false},
		{"link-local IPv6", "fe80::1", false},
		{"unspecified IPv4", "0.0.0.0", false},
		{"unspecified IPv6", "::", false},
		{"multicast", "224.0.0.1", false},
		{"link-local multicast", "224.0.0.251", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if !assert.NotNil(t, ip, "test IP %q should itself parse", tt.ip) {
				return
			}
			assert.Equal(t, tt.want, IsPublicIP(ip))
		})
	}
}

func TestIsPublicIP_Nil(t *testing.T) {
	assert.False(t, IsPublicIP(nil))
}

func TestDialControl(t *testing.T) {
	tests := []struct {
		name    string
		address string
		wantErr bool
	}{
		{"allowed public address", "8.8.8.8:443", false},
		{"blocked loopback address", "127.0.0.1:80", true},
		{"blocked link-local metadata address", "169.254.169.254:80", true},
		{"blocked private address", "10.0.0.5:5432", true},
		{"malformed address with no port", "not-an-address", true},
		{"unparseable host", "example.com:80", true}, // DialControl runs post-resolution; a bare hostname here is never a valid literal IP.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DialControl("tcp", tt.address, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
