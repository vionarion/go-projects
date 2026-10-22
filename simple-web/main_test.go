package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPeople(t *testing.T) {
	mux := newMux(&server{})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/people", nil))
	if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("empty list: %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/people", strings.NewReader(`{"name":"Ann","age":30}`)))
	if rec.Code != http.StatusCreated {
		t.Fatalf("add: %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/people", nil))
	if !strings.Contains(rec.Body.String(), `"name":"Ann"`) {
		t.Errorf("list after add: %q", rec.Body.String())
	}
}

func TestBadInput(t *testing.T) {
	mux := newMux(&server{})
	for _, body := range []string{`not json`, `{"age":5}`, `{"name":"Bob","age":-1}`} {
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/people", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: %d, want 400", body, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/people", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("DELETE: %d, want 405", rec.Code)
	}
}
