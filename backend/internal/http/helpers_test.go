package httpapi

import "testing"

func TestTimelineLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		{input: "", want: 20},
		{input: "10", want: 10},
		{input: "-1", want: 20},
		{input: "999", want: 50},
		{input: "nope", want: 20},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := timelineLimit(tc.input)
			if got != tc.want {
				t.Fatalf("timelineLimit(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsSecureCookieEnv(t *testing.T) {
	t.Parallel()

	if !isSecureCookieEnv("prod") {
		t.Fatal("expected prod env to require secure cookies")
	}
	if !isSecureCookieEnv("production") {
		t.Fatal("expected production env to require secure cookies")
	}
	if isSecureCookieEnv("dev") {
		t.Fatal("did not expect dev env to require secure cookies")
	}
}

func TestTimelineMaxID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int64
	}{
		{input: "", want: 0},
		{input: "25", want: 25},
		{input: "0", want: 0},
		{input: "-5", want: 0},
		{input: "abc", want: 0},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			got := timelineMaxID(tc.input)
			if got != tc.want {
				t.Fatalf("timelineMaxID(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsUniqueViolation(t *testing.T) {
	t.Parallel()

	if !isUniqueViolation(assertErr("duplicate key value violates unique constraint")) {
		t.Fatal("expected duplicate key error to be treated as unique violation")
	}
	if isUniqueViolation(assertErr("some other database failure")) {
		t.Fatal("did not expect generic db error to be treated as unique violation")
	}
}

func TestIsAllowedMediaType(t *testing.T) {
	t.Parallel()

	if !isAllowedMediaType("image/png") {
		t.Fatal("image/png should be allowed")
	}
	if !isAllowedMediaType("application/pdf") {
		t.Fatal("application/pdf should be allowed")
	}
	if isAllowedMediaType("application/javascript") {
		t.Fatal("application/javascript should not be allowed")
	}
}

func TestMediaURL(t *testing.T) {
	t.Parallel()

	got := mediaURL("http://localhost:8080/", "123/abc.png")
	want := "http://localhost:8080/media/123/abc.png"
	if got != want {
		t.Fatalf("mediaURL mismatch: got %q want %q", got, want)
	}
}

func assertErr(message string) error {
	return &testErr{message: message}
}

type testErr struct {
	message string
}

func (e *testErr) Error() string {
	return e.message
}
