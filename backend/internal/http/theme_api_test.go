package httpapi

import "testing"

func TestNormalizeThemePreset(t *testing.T) {
	t.Parallel()

	preset, err := normalizeThemePreset("midnight")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if preset != "midnight" {
		t.Fatalf("unexpected preset: %q", preset)
	}

	preset, err = normalizeThemePreset("")
	if err != nil {
		t.Fatalf("unexpected error for empty preset: %v", err)
	}
	if preset != "forest" {
		t.Fatalf("expected default preset forest, got %q", preset)
	}

	if _, err := normalizeThemePreset("unknown"); err == nil {
		t.Fatal("expected unknown preset to fail")
	}
}

func TestSanitizeThemeVariables(t *testing.T) {
	t.Parallel()

	vars, err := sanitizeThemeVariables(map[string]string{
		"bg":    "#abcdef",
		"paper": "#ABC",
	})
	if err != nil {
		t.Fatalf("unexpected sanitize error: %v", err)
	}
	if vars["bg"] != "#abcdef" {
		t.Fatalf("unexpected bg value: %q", vars["bg"])
	}
	if vars["paper"] != "#abc" {
		t.Fatalf("expected lowercase normalized color, got %q", vars["paper"])
	}

	if _, err := sanitizeThemeVariables(map[string]string{"oops": "#123456"}); err == nil {
		t.Fatal("expected invalid key to fail")
	}

	if _, err := sanitizeThemeVariables(map[string]string{"bg": "red"}); err == nil {
		t.Fatal("expected invalid color to fail")
	}
}
