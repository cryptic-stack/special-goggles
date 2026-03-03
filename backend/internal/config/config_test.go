package config

import "testing"

func TestExtractHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{input: "https://example.com", want: "example.com"},
		{input: "example.com:8443", want: "example.com"},
		{input: "LOCALHOST:8080", want: "localhost"},
		{input: "", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := extractHost(tc.input)
			if got != tc.want {
				t.Fatalf("extractHost(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsLocalhostLikeHost(t *testing.T) {
	t.Parallel()

	if !isLocalhostLikeHost("localhost") {
		t.Fatal("localhost should be localhost-like")
	}
	if !isLocalhostLikeHost("127.0.0.1") {
		t.Fatal("loopback IPv4 should be localhost-like")
	}
	if isLocalhostLikeHost("example.com") {
		t.Fatal("public domain should not be localhost-like")
	}
}
