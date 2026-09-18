// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// testUnitMockRoute is a swappable per-method HTTP handler; tests flip it
// between TestSteps (typically via PreConfig) to change how that method
// responds mid-test.
type testUnitMockRoute struct {
	fn atomic.Pointer[func(w http.ResponseWriter, r *http.Request)]
}

func (m *testUnitMockRoute) set(fn func(w http.ResponseWriter, r *http.Request)) {
	m.fn.Store(&fn)
}

// testUnitJSON returns a handler that writes a fixed JSON body and status.
func testUnitJSON(status int, body string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}
}

// testUnitNoContent returns a handler that writes 204 No Content.
func testUnitNoContent() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}
}

// testUnitOnceThen serves first on its first call, then on every call after.
func testUnitOnceThen(first, then func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	var used atomic.Bool
	return func(w http.ResponseWriter, r *http.Request) {
		if used.CompareAndSwap(false, true) {
			first(w, r)
			return
		}
		then(w, r)
	}
}

// testUnitNewMockServer dispatches by method to independently swappable routes.
func testUnitNewMockServer(t *testing.T, methods ...string) (baseURL string, routes []*testUnitMockRoute) {
	t.Helper()
	byMethod := make(map[string]*testUnitMockRoute, len(methods))
	routes = make([]*testUnitMockRoute, len(methods))
	for i, m := range methods {
		r := &testUnitMockRoute{}
		byMethod[m] = r
		routes[i] = r
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := byMethod[r.Method]
		var fn *func(w http.ResponseWriter, r *http.Request)
		if route != nil {
			fn = route.fn.Load()
		}
		if fn == nil {
			t.Errorf("mock server: unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		(*fn)(w, r)
	}))
	t.Cleanup(server.Close)
	return server.URL, routes
}

// testUnitMockServer starts a mock server dispatching GET/POST/PATCH/DELETE.
func testUnitMockServer(t *testing.T) (baseURL string, get, post, patch, del *testUnitMockRoute) {
	t.Helper()
	baseURL, routes := testUnitNewMockServer(t, http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete)
	return baseURL, routes[0], routes[1], routes[2], routes[3]
}

// testUnitMockServerWithPut is for resources whose Create issues a PUT.
func testUnitMockServerWithPut(t *testing.T) (baseURL string, get, post, put, patch, del *testUnitMockRoute) {
	t.Helper()
	baseURL, routes := testUnitNewMockServer(t, http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete)
	return baseURL, routes[0], routes[1], routes[2], routes[3], routes[4]
}

// testUnitProviderConfig returns a provider block pointed at a mock server.
func testUnitProviderConfig(baseURL string) string {
	return fmt.Sprintf(`
provider "ubiops" {
  api_token = "unit-test-token"
  base_url  = %[1]q
}
`, baseURL)
}
