// Package stripcookies is a Traefik middleware plugin to strip cookies by name.
package stripcookies

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Config the plugin configuration.
type Config struct {
	Cookies []string `json:"cookies,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{}
}

// StripCookies is the middleware handler.
type StripCookies struct {
	next  http.Handler
	strip map[string]struct{}
	name  string
}

// New creates a new StripCookies plugin.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.Cookies) == 0 {
		return nil, fmt.Errorf("cookies cannot be empty")
	}

	// Convert slice to map for O(1) lookups during requests.
	strip := make(map[string]struct{}, len(config.Cookies))
	for _, c := range config.Cookies {
		strip[c] = struct{}{}
	}

	return &StripCookies{
		next:  next,
		strip: strip,
		name:  name,
	}, nil
}

func (s *StripCookies) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	// Direct map access (instead of Values()) avoids CanonicalMIMEHeaderKey work.
	// "Cookie" is the canonical form set by net/http and Add/Set/Del.
	cookieHeaders := req.Header["Cookie"]
	if len(cookieHeaders) == 0 {
		s.next.ServeHTTP(rw, req)
		return
	}

	// Fast path: allocation-free scan. If nothing needs stripping,
	// leave the headers untouched (zero allocs).
	needsStrip := false

	for _, h := range cookieHeaders {
		if containsStripped(h, s.strip) {
			needsStrip = true
			break
		}
	}

	if !needsStrip {
		s.next.ServeHTTP(rw, req)
		return
	}

	// Slow path: rebuild Cookie header(s) without stripped cookies.
	// Grow once to the longest header so the builder is reused
	// across headers without re-allocating.
	maxLen := 0
	for _, h := range cookieHeaders {
		if len(h) > maxLen {
			maxLen = len(h)
		}
	}

	var b strings.Builder
	b.Grow(maxLen)

	req.Header.Del("Cookie")

	for _, h := range cookieHeaders {
		b.Reset()

		first := true
		rest := h

		// Iterate with Cut instead of Split: no intermediate []string allocation,
		// substrings share the original header backing array.
		for len(rest) > 0 {
			var pair string
			pair, rest, _ = strings.Cut(rest, ";")
			pair = strings.TrimSpace(pair)
			if pair == "" {
				continue
			}

			// Cut on first '=' only, values may contain '=' (e.g. base64 padding).
			name, _, _ := strings.Cut(pair, "=")
			if _, ok := s.strip[strings.TrimSpace(name)]; ok {
				continue
			}

			if !first {
				b.WriteString("; ")
			}
			b.WriteString(pair)
			first = false
		}

		if b.Len() > 0 {
			req.Header.Add("Cookie", b.String())
		}
	}

	s.next.ServeHTTP(rw, req)
}

// containsStripped reports whether header contains a cookie to strip.
// It performs no allocations: Cut and TrimSpace return subslices.
func containsStripped(header string, strip map[string]struct{}) bool {
	rest := header
	for len(rest) > 0 {
		var pair string
		pair, rest, _ = strings.Cut(rest, ";")
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		name, _, _ := strings.Cut(pair, "=")
		if _, ok := strip[strings.TrimSpace(name)]; ok {
			return true
		}
	}

	return false
}
