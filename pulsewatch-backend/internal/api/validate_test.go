package api

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateName(t *testing.T) {
	assert.NoError(t, validateName("Production API"))
	assert.NoError(t, validateName(""))
	assert.NoError(t, validateName(strings.Repeat("a", maxNameLength)))

	err := validateName(strings.Repeat("a", maxNameLength+1))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "200 characters or fewer")
}

func TestValidateHTTPURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr string // substring expected in the error, "" means no error
	}{
		{"valid https", "https://example.com/health", ""},
		{"valid http", "http://example.com/health", ""},
		{"valid with port and path", "https://example.com:8443/api/v1/health", ""},
		{"too long", "https://example.com/" + strings.Repeat("a", maxURLLength), "characters or fewer"},
		{"not a valid URL", "https://example.com/%", "not valid"},
		{"missing scheme", "example.com/health", "http:// or https://"},
		{"unsupported scheme", "ftp://example.com/file", "http:// or https://"},
		{"missing host", "https:///health", "must include a host"},
		{"literal localhost", "http://localhost:5432/", "local or internal address"},
		{"literal localhost mixed case", "http://LocalHost/", "local or internal address"},
		{"literal loopback IP", "http://127.0.0.1:9999/", "local or internal address"},
		{"literal private IP", "http://10.0.0.5/", "local or internal address"},
		{"literal link-local metadata IP", "http://169.254.169.254/latest/meta-data/", "local or internal address"},
		{"literal public IP is fine", "http://8.8.8.8/", ""},
		{"ordinary hostname is fine (DNS-level SSRF is the dial guard's job, not this check's)", "https://httpbin.org/status/200", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHTTPURL(tt.url)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidateStatusCode(t *testing.T) {
	assert.NoError(t, validateStatusCode(100))
	assert.NoError(t, validateStatusCode(200))
	assert.NoError(t, validateStatusCode(599))

	for _, code := range []int{0, 99, 600, -1} {
		err := validateStatusCode(code)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "between 100 and 599")
	}
}

func TestValidateMethod(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"GET as-is", "GET", "GET", false},
		{"lowercase get normalizes", "get", "GET", false},
		{"head with padding normalizes", "  HEAD  ", "HEAD", false},
		{"post", "POST", "POST", false},
		{"unsupported method", "DELETE", "", true},
		{"empty", "", "", true},
		{"garbage", "PATCH", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateMethod(tt.in)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRequireValidID(t *testing.T) {
	valid := "550e8400-e29b-41d4-a716-446655440000"
	w := httptest.NewRecorder()
	assert.True(t, requireValidID(w, valid))
	assert.Equal(t, 200, w.Code) // handler never wrote anything on the success path

	for _, id := range []string{"", "not-a-uuid", "550e8400-e29b-41d4-a716", "550e8400e29b41d4a716446655440000"} {
		w := httptest.NewRecorder()
		assert.False(t, requireValidID(w, id))
		assert.Equal(t, 404, w.Code)
		assert.Contains(t, w.Body.String(), "monitor not found")
	}
}
