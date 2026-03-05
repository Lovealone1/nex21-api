package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExtractTenantID(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		header   string
		expected string
	}{
		{
			name:     "From Subdomain",
			host:     "acme.nex21.com",
			header:   "",
			expected: "acme",
		},
		{
			name:     "From Header when no valid subdomain",
			host:     "api.nex21.com",
			header:   "stark-industries",
			expected: "stark-industries",
		},
		{
			name:     "From Header overrides generic localhost",
			host:     "localhost:8080",
			header:   "local-tenant",
			expected: "local-tenant",
		},
		{
			name:     "Subdomain ignores www",
			host:     "www.stark.nex21.com",
			header:   "",
			expected: "stark",
		},
		{
			name:     "No Tenant Provided",
			host:     "localhost:8080",
			header:   "",
			expected: "",
		},
		{
			name:     "Normalization (Uppercase to Lowercase)",
			host:     "ACME.nex21.com",
			header:   "",
			expected: "acme",
		},
		{
			name:     "Normalization (Header Whitespace and Case)",
			host:     "localhost",
			header:   "  WaYNE-Ent  ",
			expected: "wayne-ent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://"+tt.host+"/api/v1/test", nil)
			if tt.header != "" {
				req.Header.Set("X-Tenant-ID", tt.header)
			}

			// Some subdomain setups could misplace Host, ensure it's set
			req.Host = tt.host

			actual := ExtractTenantID(req)
			if actual != tt.expected {
				t.Errorf("ExtractTenantID() = %q, want %q", actual, tt.expected)
			}
		})
	}
}
