// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testUnitServiceDataSourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
data "ubiops_service" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-service"
}
`
}

func TestUnitServiceDataSource_ReadError(t *testing.T) {
	baseURL, get, _, _, _ := testUnitMockServer(t)
	get.set(testUnitJSON(500, `{"error":"internal error"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitServiceDataSourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error reading service`),
			},
		},
	})
}
