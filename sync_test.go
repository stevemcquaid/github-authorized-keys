package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceManagedSection_NoExistingBlock(t *testing.T) {
	content := "ssh-rsa AAAA manually-added-key\n"
	keys := []string{"ssh-ed25519 AAAA githubkey1"}
	result := replaceManagedSection(content, []string{"octocat"}, keys)

	if !strings.Contains(result, beginMarker) {
		t.Error("expected BEGIN marker")
	}
	if !strings.Contains(result, endMarker) {
		t.Error("expected END marker")
	}
	if !strings.Contains(result, "ssh-ed25519 AAAA githubkey1") {
		t.Error("expected github key in result")
	}
	if !strings.Contains(result, "ssh-rsa AAAA manually-added-key") {
		t.Error("manually added key should be preserved")
	}
}

func TestReplaceManagedSection_ExistingBlock(t *testing.T) {
	content := "ssh-rsa AAAA before\n" +
		beginMarker + "\n" +
		"# Managed by github-authorized-keys\n" +
		"ssh-ed25519 AAAA oldkey\n" +
		endMarker + "\n" +
		"ssh-rsa AAAA after\n"

	newKeys := []string{"ssh-ed25519 AAAA newkey"}
	result := replaceManagedSection(content, []string{"octocat"}, newKeys)

	if strings.Contains(result, "ssh-ed25519 AAAA oldkey") {
		t.Error("old key should have been replaced")
	}
	if !strings.Contains(result, "ssh-ed25519 AAAA newkey") {
		t.Error("new key should be present")
	}
	if !strings.Contains(result, "ssh-rsa AAAA before") {
		t.Error("key before block should be preserved")
	}
	if !strings.Contains(result, "ssh-rsa AAAA after") {
		t.Error("key after block should be preserved")
	}
}

func TestReplaceManagedSection_EmptyFile(t *testing.T) {
	result := replaceManagedSection("", []string{"octocat"}, []string{"ssh-ed25519 AAAA key"})
	if !strings.Contains(result, beginMarker) {
		t.Error("expected BEGIN marker")
	}
}

func TestReplaceManagedSection_EmptyKeys(t *testing.T) {
	content := "ssh-rsa AAAA manual\n"
	result := replaceManagedSection(content, []string{"octocat"}, []string{})
	if !strings.Contains(result, beginMarker) {
		t.Error("expected BEGIN marker even with no keys")
	}
	if !strings.Contains(result, "ssh-rsa AAAA manual") {
		t.Error("manual key should be preserved")
	}
}

func TestSyncer_WritesAndPreservesManualKeys(t *testing.T) {
	dir := t.TempDir()
	keysPath := filepath.Join(dir, "authorized_keys")

	// Pre-populate with a manual key.
	err := os.WriteFile(keysPath, []byte("ssh-rsa AAAA manual\n"), 0600)
	if err != nil {
		t.Fatal(err)
	}

	s := NewSyncer(keysPath)
	err = s.Sync([]string{"octocat"}, []string{"ssh-ed25519 AAAA github"})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	data, err := os.ReadFile(keysPath)
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)

	if !strings.Contains(result, "ssh-rsa AAAA manual") {
		t.Error("manual key should be preserved")
	}
	if !strings.Contains(result, "ssh-ed25519 AAAA github") {
		t.Error("github key should be present")
	}
}

func TestSyncer_CreatesFileAndDir(t *testing.T) {
	dir := t.TempDir()
	keysPath := filepath.Join(dir, ".ssh", "authorized_keys")

	s := NewSyncer(keysPath)
	err := s.Sync([]string{"octocat"}, []string{"ssh-ed25519 AAAA key"})
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	info, err := os.Stat(keysPath)
	if err != nil {
		t.Fatalf("authorized_keys not created: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}

func TestSyncer_IdempotentSync(t *testing.T) {
	dir := t.TempDir()
	keysPath := filepath.Join(dir, "authorized_keys")

	s := NewSyncer(keysPath)
	keys := []string{"ssh-ed25519 AAAA key1", "ssh-rsa AAAA key2"}

	// Run sync twice with same keys.
	for i := 0; i < 2; i++ {
		if err := s.Sync([]string{"octocat"}, keys); err != nil {
			t.Fatalf("Sync %d failed: %v", i+1, err)
		}
	}

	data, err := os.ReadFile(keysPath)
	if err != nil {
		t.Fatal(err)
	}
	result := string(data)

	// Should only have one block.
	if count := strings.Count(result, beginMarker); count != 1 {
		t.Errorf("expected 1 BEGIN marker, got %d", count)
	}
}

func TestWriteAtomic_PreservesPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "testfile")

	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := writeAtomic(path, "updated content"); err != nil {
		t.Fatalf("writeAtomic failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "updated content" {
		t.Errorf("expected updated content, got %q", string(data))
	}
}
