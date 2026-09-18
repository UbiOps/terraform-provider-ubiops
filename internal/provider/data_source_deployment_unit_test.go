// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testUnitDeploymentDataSourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
data "ubiops_deployment" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-deployment"
}
`
}

func TestUnitDeploymentDataSource_ReadError(t *testing.T) {
	baseURL, get, _, _, _ := testUnitMockServer(t)
	get.set(testUnitJSON(404, `{"error":"deployment not found"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitDeploymentDataSourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error reading deployment`),
			},
		},
	})
}
