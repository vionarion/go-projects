package api

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"github.com/vionarion/go-projects/url-shortener/internal/store"
)

func newTestApp(t *testing.T, quota int) (*fiber.App, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr(), MaxRetries: -1, DialTimeout: 200 * time.Millisecond})
	app := New(Config{
		Domain:     "s.example.com",
		Quota:      quota,
		Window:     30 * time.Minute,
		DefaultTTL: 24 * time.Hour,
		MaxTTL:     48 * time.Hour,
		CodeLength: 7,
	}, store.NewRedis(rdb))
	return app, mr
}

func shorten(t *testing.T, app *fiber.App, body string) (*http.Response, map[string]any) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	data, _ := io.ReadAll(resp.Body)
	_ = json.Unmarshal(data, &out)
	return resp, out
}

func TestShortenAndRedirect(t *testing.T) {
	app, _ := newTestApp(t, 10)

	resp, out := shorten(t, app, `{"url":"example.org/page?a=1"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status %d, body %v", resp.StatusCode, out)
	}
	if out["url"] != "https://example.org/page?a=1" {
		t.Errorf("url not normalized: %v", out["url"])
	}
	short := out["short_url"].(string)
	if !strings.HasPrefix(short, "http://s.example.com/") {
		t.Fatalf("unexpected short url %q", short)
	}

	code := short[strings.LastIndex(short, "/")+1:]
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/"+code, nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusFound || resp.Header.Get("Location") != "https://example.org/page?a=1" {
		t.Errorf("redirect: status %d, location %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestCustomCode(t *testing.T) {
	app, _ := newTestApp(t, 10)

	resp, _ := shorten(t, app, `{"url":"https://go.dev","code":"golang"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("first: status %d", resp.StatusCode)
	}
	resp, _ = shorten(t, app, `{"url":"https://go.dev/doc","code":"golang"}`)
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("taken code: status %d, want 409", resp.StatusCode)
	}
	resp, _ = shorten(t, app, `{"url":"https://go.dev","code":"a b"}`)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad code: status %d, want 400", resp.StatusCode)
	}
}

func TestRejectsBadURLs(t *testing.T) {
	app, _ := newTestApp(t, 100)
	for _, body := range []string{
		`{"url":""}`,
		`{"url":"ftp://example.org"}`,
		`{"url":"localhost"}`,
		`{"url":"https://www.s.example.com/abc"}`, // own domain: no redirect loops
		`not json`,
	} {
		if resp, out := shorten(t, app, body); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d (%v), want 400", body, resp.StatusCode, out)
		}
	}
}

func TestExpiry(t *testing.T) {
	app, mr := newTestApp(t, 10)

	_, out := shorten(t, app, `{"url":"https://example.org","code":"short","expiry_hours":1000}`)
	if out["expires_in_hours"] != float64(48) {
		t.Errorf("expiry not capped: %v", out["expires_in_hours"])
	}
	mr.FastForward(49 * time.Hour)
	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/short", nil))
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expired link: status %d, want 404", resp.StatusCode)
	}
}

func TestRateLimit(t *testing.T) {
	app, mr := newTestApp(t, 2)

	for i := 0; i < 2; i++ {
		if resp, _ := shorten(t, app, `{"url":"https://example.org"}`); resp.StatusCode != http.StatusCreated {
			t.Fatalf("request %d: status %d", i, resp.StatusCode)
		}
	}
	resp, _ := shorten(t, app, `{"url":"https://example.org"}`)
	if resp.StatusCode != http.StatusTooManyRequests || resp.Header.Get("Retry-After") == "" {
		t.Fatalf("over quota: status %d, Retry-After %q", resp.StatusCode, resp.Header.Get("Retry-After"))
	}

	mr.FastForward(31 * time.Minute)
	if resp, _ := shorten(t, app, `{"url":"https://example.org"}`); resp.StatusCode != http.StatusCreated {
		t.Errorf("after window: status %d", resp.StatusCode)
	}
}

func TestMetrics(t *testing.T) {
	app, _ := newTestApp(t, 10)

	resp, out := shorten(t, app, `{"url":"https://example.org"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("shorten: status %d, body %v", resp.StatusCode, out)
	}
	code := out["short_url"].(string)
	code = code[strings.LastIndex(code, "/")+1:]
	if r, err := app.Test(httptest.NewRequest(http.MethodGet, "/"+code, nil)); err != nil || r.StatusCode != http.StatusFound {
		t.Fatalf("redirect: status %v, err %v", r, err)
	}

	after, _ := app.Test(httptest.NewRequest(http.MethodGet, "/metrics", nil))
	afterBody, _ := io.ReadAll(after.Body)
	body := string(afterBody)
	if !strings.Contains(body, "shortener_links_created_total 1") {
		t.Errorf("links created counter did not increase: %s", body)
	}
	if !strings.Contains(body, "shortener_redirects_total 1") {
		t.Errorf("redirects counter did not increase: %s", body)
	}
	if !strings.Contains(body, `route="/:code"`) {
		t.Errorf("requests should be labeled by route template, not raw path: %s", body)
	}
}

func TestHealth(t *testing.T) {
	app, mr := newTestApp(t, 10)

	resp, _ := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil))
	if resp.StatusCode != http.StatusOK {
		t.Errorf("health: status %d", resp.StatusCode)
	}
	mr.Close()
	resp, err := app.Test(httptest.NewRequest(http.MethodGet, "/health", nil), -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("health without redis: status %d, want 503", resp.StatusCode)
	}
}
