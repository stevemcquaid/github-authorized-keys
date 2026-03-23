package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchKeys_SingleUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA testkey1\nssh-rsa AAAAB3Nza testkey2\n"))
	}))
	defer server.Close()

	f := &Fetcher{client: patchedClient(server.URL)}
	keys, err := f.FetchKeys([]string{"testuser"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

func TestFetchKeys_Deduplication(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusOK)
		// Both users return the same key
		_, _ = w.Write([]byte("ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA sharedkey\n"))
	}))
	defer server.Close()

	f := &Fetcher{client: patchedClient(server.URL)}
	keys, err := f.FetchKeys([]string{"user1", "user2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 deduplicated key, got %d", len(keys))
	}
	if calls != 2 {
		t.Errorf("expected 2 HTTP calls, got %d", calls)
	}
}

func TestFetchKeys_404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	f := &Fetcher{client: patchedClient(server.URL)}
	_, err := f.FetchKeys([]string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestFetchKeys_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("\n\n\n"))
	}))
	defer server.Close()

	f := &Fetcher{client: patchedClient(server.URL)}
	keys, err := f.FetchKeys([]string{"nokeys"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys, got %d", len(keys))
	}
}

func TestParseKeys(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "normal keys",
			input:    "ssh-ed25519 AAAA key1\nssh-rsa AAAA key2\n",
			expected: []string{"ssh-ed25519 AAAA key1", "ssh-rsa AAAA key2"},
		},
		{
			name:     "blank lines stripped",
			input:    "\nssh-ed25519 AAAA key1\n\n  \n",
			expected: []string{"ssh-ed25519 AAAA key1"},
		},
		{
			name:     "empty body",
			input:    "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseKeys(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("expected %d keys, got %d", len(tt.expected), len(got))
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("key[%d]: expected %q, got %q", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

// patchedClient returns an http.Client whose transport rewrites requests to the
// mock server URL so we can use the real Fetcher code against a test server.
func patchedClient(serverURL string) *http.Client {
	return &http.Client{
		Transport: &rewriteTransport{base: http.DefaultTransport, serverURL: serverURL},
	}
}

type rewriteTransport struct {
	base      http.RoundTripper
	serverURL string
}

func (rt *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect all requests to the test server, preserving path.
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = rt.serverURL[len("http://"):]
	return rt.base.RoundTrip(req2)
}
