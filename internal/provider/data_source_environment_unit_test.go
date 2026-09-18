// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testUnitEnvironmentDataSourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
data "ubiops_environment" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-environment"
}
`
}

func TestUnitEnvironmentDataSource_ReadError(t *testing.T) {
	baseURL, get, _, _, _ := testUnitMockServer(t)
	get.set(testUnitJSON(404, `{"error":"environment not found"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitEnvironmentDataSourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error reading environment`),
			},
		},
	})
}
