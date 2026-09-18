// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
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

// Create on this resource issues a PUT.
func TestUnitServiceUserTokenResource_CreateError(t *testing.T) {
	baseURL, _, _, put, _, _ := testUnitMockServerWithPut(t)
	put.set(testUnitJSON(400, `{"error":"invalid service user token request"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitServiceUserTokenResourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error creating service user token`),
			},
		},
	})
}
