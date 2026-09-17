// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testUnitServiceUserTokenResourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_service_user_token" "test" {
  project_name    = "unit-test-project"
  service_user_id = "33333333-3333-3333-3333-333333333333"
}
`
}

// The shared testUnitMockServer only routes GET/POST/PATCH/DELETE. Create on
// this resource issues a PUT, so it uses its own minimal PUT-only server.
func TestUnitServiceUserTokenResource_CreateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("mock server: unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(400)
		_, _ = io.WriteString(w, `{"error":"invalid service user token request"}`)
	}))
	t.Cleanup(server.Close)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitServiceUserTokenResourceConfig(server.URL),
				ExpectError: regexp.MustCompile(`Error creating service user token`),
			},
		},
	})
}
