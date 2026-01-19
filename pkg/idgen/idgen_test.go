package idgen

import (
	"regexp"
	"testing"
)

func TestGenerate_Format(t *testing.T) {
	// ID should match format "ar-xxxx" where xxxx is 4 hex characters
	pattern := regexp.MustCompile(`^ar-[0-9a-f]{4}$`)

	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if !pattern.MatchString(id) {
		t.Errorf("Generate() = %v, want format ar-[0-9a-f]{4}", id)
	}
}

func TestGenerate_Unique(t *testing.T) {
	// Generate multiple IDs and verify they are unique
	ids := make(map[string]bool)
	count := 100

	for i := 0; i < count; i++ {
		id, err := Generate()
		if err != nil {
			t.Fatalf("Generate() returned error: %v", err)
		}
		if ids[id] {
			t.Errorf("Generate() returned duplicate ID: %v", id)
		}
		ids[id] = true
	}
}

func TestGenerate_ValidCharacters(t *testing.T) {
	// Generate multiple IDs and verify they only contain expected characters
	validChars := regexp.MustCompile(`^[ar0-9a-f-]+$`)

	for i := 0; i < 50; i++ {
		id, err := Generate()
		if err != nil {
			t.Fatalf("Generate() returned error: %v", err)
		}
		if !validChars.MatchString(id) {
			t.Errorf("Generate() = %v contains invalid characters", id)
		}
	}
}

func TestGenerate_Prefix(t *testing.T) {
	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if len(id) < 3 || id[:3] != "ar-" {
		t.Errorf("Generate() = %v, should start with 'ar-'", id)
	}
}

func TestGenerate_Length(t *testing.T) {
	// ar-xxxx = 7 characters total (2 + 1 + 4)
	expectedLen := 7

	id, err := Generate()
	if err != nil {
		t.Fatalf("Generate() returned error: %v", err)
	}

	if len(id) != expectedLen {
		t.Errorf("Generate() length = %d, want %d", len(id), expectedLen)
	}
}

func TestMustGenerate_Format(t *testing.T) {
	pattern := regexp.MustCompile(`^ar-[0-9a-f]{4}$`)

	id := MustGenerate()

	if !pattern.MatchString(id) {
		t.Errorf("MustGenerate() = %v, want format ar-[0-9a-f]{4}", id)
	}
}

func TestMustGenerate_DoesNotPanic(t *testing.T) {
	// Under normal circumstances, MustGenerate should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustGenerate() panicked: %v", r)
		}
	}()

	_ = MustGenerate()
}

func TestPrefix_Constant(t *testing.T) {
	if Prefix != "ar" {
		t.Errorf("Prefix = %v, want 'ar'", Prefix)
	}
}

func TestIDLength_Constant(t *testing.T) {
	if IDLength != 4 {
		t.Errorf("IDLength = %d, want 4", IDLength)
	}
}
