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

// testUnitMockServer starts a local HTTP server standing in for the UbiOps
// API, dispatching by method to independently swappable routes. Unset
// methods 500 with a test failure, surfacing any call the test didn't expect.
func testUnitMockServer(t *testing.T) (baseURL string, get, post, patch, del *testUnitMockRoute) {
	t.Helper()
	get, post, patch, del = &testUnitMockRoute{}, &testUnitMockRoute{}, &testUnitMockRoute{}, &testUnitMockRoute{}

	routeFor := func(method string) *testUnitMockRoute {
		switch method {
		case http.MethodGet:
			return get
		case http.MethodPost:
			return post
		case http.MethodPatch:
			return patch
		case http.MethodDelete:
			return del
		default:
			return nil
		}
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := routeFor(r.Method)
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
	return server.URL, get, post, patch, del
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
