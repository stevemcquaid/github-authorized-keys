package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	githubKeysURL  = "https://github.com/%s.keys"
	fetchTimeout   = 10 * time.Second
)

// Fetcher retrieves SSH public keys from GitHub profiles.
type Fetcher struct {
	client *http.Client
}

// NewFetcher creates a Fetcher with a sensible default HTTP client.
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: fetchTimeout},
	}
}

// FetchKeys fetches SSH public keys for all given GitHub usernames and
// returns a deduplicated, ordered slice of key strings.
func (f *Fetcher) FetchKeys(usernames []string) ([]string, error) {
	seen := make(map[string]struct{})
	var keys []string

	for _, username := range usernames {
		userKeys, err := f.fetchUser(username)
		if err != nil {
			return nil, fmt.Errorf("fetching keys for %s: %w", username, err)
		}
		for _, k := range userKeys {
			if _, dup := seen[k]; !dup {
				seen[k] = struct{}{}
				keys = append(keys, k)
			}
		}
	}
	return keys, nil
}

// fetchUser fetches the public keys for a single GitHub username.
func (f *Fetcher) fetchUser(username string) ([]string, error) {
	url := fmt.Sprintf(githubKeysURL, username)
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("GitHub user %q not found (404)", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d fetching keys for %q", resp.StatusCode, username)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	return parseKeys(string(body)), nil
}

// parseKeys splits a newline-delimited key blob into individual key strings,
// stripping blank lines and surrounding whitespace.
func parseKeys(body string) []string {
	var keys []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			keys = append(keys, line)
		}
	}
	return keys
}
