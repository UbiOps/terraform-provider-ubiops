// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testUnitProjectDataSourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
data "ubiops_project" "test" {
  name = "unit-test-project"
}
`
}

func TestUnitProjectDataSource_ReadError(t *testing.T) {
	baseURL, get, _, _, _ := testUnitMockServer(t)
	get.set(testUnitJSON(500, `{"error":"internal error"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitProjectDataSourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error reading project`),
			},
		},
	})
}
