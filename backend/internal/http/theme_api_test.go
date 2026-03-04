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
	if preset != "gnusocial" {
		t.Fatalf("expected default preset gnusocial, got %q", preset)
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

func TestSanitizeThemeOptions(t *testing.T) {
	t.Parallel()

	opts, err := sanitizeThemeOptions(map[string]string{
		"font":    "mono",
		"density": "compact",
		"corner":  "sharp",
	})
	if err != nil {
		t.Fatalf("unexpected sanitize options error: %v", err)
	}
	if opts["font"] != "mono" || opts["density"] != "compact" || opts["corner"] != "sharp" {
		t.Fatalf("unexpected options payload: %#v", opts)
	}

	defaulted, err := sanitizeThemeOptions(nil)
	if err != nil {
		t.Fatalf("unexpected default sanitize options error: %v", err)
	}
	if defaulted["font"] != "modern" || defaulted["density"] != "comfortable" || defaulted["corner"] != "soft" {
		t.Fatalf("unexpected default options: %#v", defaulted)
	}

	if _, err := sanitizeThemeOptions(map[string]string{"nope": "x"}); err == nil {
		t.Fatal("expected invalid key to fail")
	}
	if _, err := sanitizeThemeOptions(map[string]string{"font": "comic"}); err == nil {
		t.Fatal("expected invalid value to fail")
	}
}
