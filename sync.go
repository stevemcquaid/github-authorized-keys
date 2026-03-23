package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	beginMarker = "# BEGIN github-authorized-keys"
	endMarker   = "# END github-authorized-keys"
)

// Syncer manages the authorized_keys file.
type Syncer struct {
	keysPath string
}

// NewSyncer creates a Syncer for the given authorized_keys path.
func NewSyncer(keysPath string) *Syncer {
	return &Syncer{keysPath: keysPath}
}

// Sync replaces the managed section of authorized_keys with the given keys.
// Lines outside the managed block are preserved unchanged.
func (s *Syncer) Sync(usernames []string, keys []string) error {
	if err := ensureSSHDir(s.keysPath); err != nil {
		return err
	}

	existing, err := readFile(s.keysPath)
	if err != nil {
		return err
	}

	updated := replaceManagedSection(existing, usernames, keys)
	return writeAtomic(s.keysPath, updated)
}

// replaceManagedSection inserts or replaces the managed block within content.
func replaceManagedSection(content string, usernames []string, keys []string) string {
	block := buildBlock(usernames, keys)

	begin := strings.Index(content, beginMarker)
	end := strings.Index(content, endMarker)

	if begin == -1 || end == -1 || end < begin {
		// No existing block — append.
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		return content + block
	}

	// Replace everything from beginMarker to end of endMarker line.
	endLineEnd := end + len(endMarker)
	if endLineEnd < len(content) && content[endLineEnd] == '\n' {
		endLineEnd++
	}

	return content[:begin] + block + content[endLineEnd:]
}

// buildBlock constructs the managed section string.
func buildBlock(usernames []string, keys []string) string {
	var b strings.Builder
	b.WriteString(beginMarker + "\n")
	b.WriteString("# Managed by github-authorized-keys — do not edit this block manually\n")
	fmt.Fprintf(&b, "# Last synced: %s — source: %s\n",
		time.Now().UTC().Format(time.RFC3339),
		strings.Join(usernames, ", "),
	)
	for _, k := range keys {
		b.WriteString(k + "\n")
	}
	b.WriteString(endMarker + "\n")
	return b.String()
}

// ensureSSHDir creates the .ssh directory and authorized_keys file if absent.
func ensureSSHDir(keysPath string) error {
	dir := filepath.Dir(keysPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}
	// Create the file if it doesn't exist yet.
	f, err := os.OpenFile(keysPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("ensuring %s exists: %w", keysPath, err)
	}
	return f.Close()
}

// readFile reads the file at path, returning "" if it doesn't exist.
func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return string(data), nil
}

// writeAtomic writes content to path via a temporary file then renames.
// The temp file is placed in the same directory to ensure same-filesystem rename.
func writeAtomic(path string, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".authorized_keys_tmp_")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up temp file on any error.
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("setting temp file permissions: %w", err)
	}

	if _, err := tmp.WriteString(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("renaming temp file to %s: %w", path, err)
	}

	success = true
	return nil
}
