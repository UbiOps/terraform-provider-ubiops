// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testUnitInstanceTypeGroupDataSourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
data "ubiops_instance_type_group" "test" {
  project_name = "unit-test-project"
  id           = "11111111-1111-1111-1111-111111111111"
}
`
}

func TestUnitInstanceTypeGroupDataSource_ReadError(t *testing.T) {
	baseURL, get, _, _, _ := testUnitMockServer(t)
	get.set(testUnitJSON(500, `{"error":"internal error"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitInstanceTypeGroupDataSourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error reading instance type group`),
			},
		},
	})
}
