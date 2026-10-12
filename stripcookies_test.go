package stripcookies_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	stripcookies "github.com/telecomnancy-net/strip-cookies"
)

func TestNewFailsOnEmptyCookies(t *testing.T) {
	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	if _, err := stripcookies.New(ctx, next, stripcookies.CreateConfig(), "strip-cookies"); err == nil {
		t.Fatal("expected error for empty cookies, got nil")
	}
}

func TestStripCookies(t *testing.T) {
	tests := []struct {
		name     string
		strip    []string
		cookies  []string
		expected []string
	}{
		{
			name:     "strip one of many",
			strip:    []string{"testCookie"},
			cookies:  []string{"testCookie=testValue; testCookie2=testValue2"},
			expected: []string{"testCookie2=testValue2"},
		},
		{
			name:     "strip multiple across headers",
			strip:    []string{"testCookie", "otherCookie"},
			cookies:  []string{"testCookie=testValue; testCookie2=testValue2", "testCookie=testValue; otherCookie=otherValue==abc"},
			expected: []string{"testCookie2=testValue2"},
		},
		{
			name:     "value containing equals is preserved",
			strip:    []string{"otherCookie"},
			cookies:  []string{"keep=otherValue==abc; otherCookie=x"},
			expected: []string{"keep=otherValue==abc"},
		},
		{
			name:     "all stripped removes header",
			strip:    []string{"a", "b"},
			cookies:  []string{"a=1; b=2"},
			expected: nil,
		},
		{
			name:     "nothing to strip leaves header untouched",
			strip:    []string{"missing"},
			cookies:  []string{"a=1; b=2"},
			expected: []string{"a=1; b=2"},
		},
		{
			name:     "tolerates extra separators and spaces",
			strip:    []string{"b"},
			cookies:  []string{"  a=1 ;;  b=2 ; ; c=3  "},
			expected: []string{"a=1; c=3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := stripcookies.CreateConfig()
			cfg.Cookies = tt.strip

			ctx := context.Background()
			next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

			handler, err := stripcookies.New(ctx, next, cfg, "strip-cookies")
			if err != nil {
				t.Fatal(err)
			}

			req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
			for _, c := range tt.cookies {
				req.Header.Add("Cookie", c)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			got := req.Header["Cookie"]
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d cookie header(s) %q, got %d %q", len(tt.expected), tt.expected, len(got), got)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("header %d: expected %q, got %q", i, tt.expected[i], got[i])
				}
			}
		})
	}
}

func TestNoCookieHeaderPassthrough(t *testing.T) {
	cfg := stripcookies.CreateConfig()
	cfg.Cookies = []string{"a"}

	handler, err := stripcookies.New(context.Background(), http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}), cfg, "strip-cookies")
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)

	if _, ok := req.Header["Cookie"]; ok {
		t.Errorf("expected no Cookie header, got %q", req.Header["Cookie"])
	}
}

func BenchmarkServeHTTPStrip(b *testing.B) {
	cfg := stripcookies.CreateConfig()
	cfg.Cookies = []string{"session_id", "tracking_id"}

	middleware, err := stripcookies.New(context.Background(), http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}), cfg, "strip-cookies")
	if err != nil {
		b.Fatal(err)
	}

	const header = "user_pref=dark; session_id=123456789; tracking_id=abcdef; lang=en-US"
	rw := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
		req.Header["Cookie"] = []string{header}
		middleware.ServeHTTP(rw, req)
	}
}

func BenchmarkServeHTTPNoOp(b *testing.B) {
	cfg := stripcookies.CreateConfig()
	cfg.Cookies = []string{"missing"}

	middleware, err := stripcookies.New(context.Background(), http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {}), cfg, "strip-cookies")
	if err != nil {
		b.Fatal(err)
	}

	const header = "user_pref=dark; lang=en-US"
	rw := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
		req.Header["Cookie"] = []string{header}
		middleware.ServeHTTP(rw, req)
	}
}
