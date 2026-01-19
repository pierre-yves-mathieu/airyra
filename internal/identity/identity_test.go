package identity

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestIdentity_FullFormat verifies output matches `user@hostname:/path` format
func TestIdentity_FullFormat(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		hostname string
		cwd      string
		want     string
	}{
		{
			name:     "basic format",
			user:     "alice",
			hostname: "macbook",
			cwd:      "/Users/alice/projects/myapp",
			want:     "alice@macbook:/Users/alice/projects/myapp",
		},
		{
			name:     "dev server example",
			user:     "dev",
			hostname: "server",
			cwd:      "/home/dev/backend",
			want:     "dev@server:/home/dev/backend",
		},
		{
			name:     "root path",
			user:     "root",
			hostname: "host",
			cwd:      "/",
			want:     "root@host:/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWithOverrides(tt.user, tt.hostname, tt.cwd)
			if got != tt.want {
				t.Errorf("GenerateWithOverrides() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestIdentity_FallbackUser verifies "unknown" is used when user is empty
func TestIdentity_FallbackUser(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		hostname string
		cwd      string
		wantUser string
	}{
		{
			name:     "empty user",
			user:     "",
			hostname: "host",
			cwd:      "/path",
			wantUser: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWithOverrides(tt.user, tt.hostname, tt.cwd)
			if !strings.HasPrefix(got, tt.wantUser+"@") {
				t.Errorf("GenerateWithOverrides() = %q, want user to be %q", got, tt.wantUser)
			}
		})
	}
}

// TestIdentity_FallbackHostname verifies "localhost" is used when hostname is empty
func TestIdentity_FallbackHostname(t *testing.T) {
	tests := []struct {
		name         string
		user         string
		hostname     string
		cwd          string
		wantHostname string
	}{
		{
			name:         "empty hostname",
			user:         "user",
			hostname:     "",
			cwd:          "/path",
			wantHostname: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWithOverrides(tt.user, tt.hostname, tt.cwd)
			expected := "@" + tt.wantHostname + ":"
			if !strings.Contains(got, expected) {
				t.Errorf("GenerateWithOverrides() = %q, want hostname to be %q", got, tt.wantHostname)
			}
		})
	}
}

// TestIdentity_FallbackCwd verifies "." is used when cwd is empty
func TestIdentity_FallbackCwd(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		hostname string
		cwd     string
		wantCwd string
	}{
		{
			name:     "empty cwd",
			user:     "user",
			hostname: "host",
			cwd:      "",
			wantCwd:  ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWithOverrides(tt.user, tt.hostname, tt.cwd)
			if !strings.HasSuffix(got, ":"+tt.wantCwd) {
				t.Errorf("GenerateWithOverrides() = %q, want cwd to be %q", got, tt.wantCwd)
			}
		})
	}
}

// TestIdentity_SpecialCharactersInPath handles paths with spaces and special chars
func TestIdentity_SpecialCharactersInPath(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		hostname string
		cwd      string
		want     string
	}{
		{
			name:     "path with spaces",
			user:     "user",
			hostname: "host",
			cwd:      "/home/user/My Projects/app",
			want:     "user@host:/home/user/My Projects/app",
		},
		{
			name:     "path with special characters",
			user:     "user",
			hostname: "host",
			cwd:      "/home/user/project-name_v2.0",
			want:     "user@host:/home/user/project-name_v2.0",
		},
		{
			name:     "path with unicode",
			user:     "user",
			hostname: "host",
			cwd:      "/home/user/projekty",
			want:     "user@host:/home/user/projekty",
		},
		{
			name:     "path with parentheses",
			user:     "user",
			hostname: "host",
			cwd:      "/home/user/project (copy)",
			want:     "user@host:/home/user/project (copy)",
		},
		{
			name:     "path with ampersand",
			user:     "user",
			hostname: "host",
			cwd:      "/home/user/foo&bar",
			want:     "user@host:/home/user/foo&bar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWithOverrides(tt.user, tt.hostname, tt.cwd)
			if got != tt.want {
				t.Errorf("GenerateWithOverrides() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestIdentity_Deterministic verifies same inputs produce same identity
func TestIdentity_Deterministic(t *testing.T) {
	user := "testuser"
	hostname := "testhost"
	cwd := "/test/path"

	// Generate multiple times and verify consistency
	results := make([]string, 10)
	for i := 0; i < 10; i++ {
		results[i] = GenerateWithOverrides(user, hostname, cwd)
	}

	first := results[0]
	for i, r := range results {
		if r != first {
			t.Errorf("GenerateWithOverrides() not deterministic: result[0] = %q, result[%d] = %q", first, i, r)
		}
	}
}

// TestIdentity_GenerateFormat verifies Generate() returns proper format
func TestIdentity_GenerateFormat(t *testing.T) {
	identity := Generate()

	// Should match pattern: user@hostname:/path
	pattern := `^[^@]+@[^:]+:.+$`
	matched, err := regexp.MatchString(pattern, identity)
	if err != nil {
		t.Fatalf("regex error: %v", err)
	}
	if !matched {
		t.Errorf("Generate() = %q, does not match expected pattern %q", identity, pattern)
	}

	// Verify it contains the @ and : separators
	if !strings.Contains(identity, "@") {
		t.Errorf("Generate() = %q, missing @ separator", identity)
	}
	if !strings.Contains(identity, ":") {
		t.Errorf("Generate() = %q, missing : separator", identity)
	}

	// The part after : should be a path (starts with / or .)
	colonIdx := strings.LastIndex(identity, ":")
	if colonIdx == -1 {
		t.Fatalf("Generate() = %q, no colon found", identity)
	}
	path := identity[colonIdx+1:]
	if !strings.HasPrefix(path, "/") && path != "." {
		t.Errorf("Generate() path part = %q, should start with / or be .", path)
	}
}

// TestIdentity_GenerateWithEnvOverride tests Generate with USER env var manipulation
func TestIdentity_GenerateWithEnvOverride(t *testing.T) {
	// Save original USER env var
	originalUser := os.Getenv("USER")
	defer os.Setenv("USER", originalUser)

	// Test with custom USER
	os.Setenv("USER", "testenvuser")
	identity := Generate()

	if !strings.HasPrefix(identity, "testenvuser@") {
		t.Errorf("Generate() = %q, expected to start with 'testenvuser@'", identity)
	}
}

// TestIdentity_AllFallbacks tests when all values need fallbacks
func TestIdentity_AllFallbacks(t *testing.T) {
	got := GenerateWithOverrides("", "", "")
	want := "unknown@localhost:."
	if got != want {
		t.Errorf("GenerateWithOverrides(\"\", \"\", \"\") = %q, want %q", got, want)
	}
}

// TestIdentity_MixedFallbacks tests various combinations of fallbacks
func TestIdentity_MixedFallbacks(t *testing.T) {
	tests := []struct {
		name     string
		user     string
		hostname string
		cwd      string
		want     string
	}{
		{
			name:     "only user missing",
			user:     "",
			hostname: "myhost",
			cwd:      "/path",
			want:     "unknown@myhost:/path",
		},
		{
			name:     "only hostname missing",
			user:     "myuser",
			hostname: "",
			cwd:      "/path",
			want:     "myuser@localhost:/path",
		},
		{
			name:     "only cwd missing",
			user:     "myuser",
			hostname: "myhost",
			cwd:      "",
			want:     "myuser@myhost:.",
		},
		{
			name:     "user and hostname missing",
			user:     "",
			hostname: "",
			cwd:      "/path",
			want:     "unknown@localhost:/path",
		},
		{
			name:     "user and cwd missing",
			user:     "",
			hostname: "myhost",
			cwd:      "",
			want:     "unknown@myhost:.",
		},
		{
			name:     "hostname and cwd missing",
			user:     "myuser",
			hostname: "",
			cwd:      "",
			want:     "myuser@localhost:.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateWithOverrides(tt.user, tt.hostname, tt.cwd)
			if got != tt.want {
				t.Errorf("GenerateWithOverrides(%q, %q, %q) = %q, want %q",
					tt.user, tt.hostname, tt.cwd, got, tt.want)
			}
		})
	}
}
