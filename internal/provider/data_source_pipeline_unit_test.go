// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testUnitPipelineDataSourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
data "ubiops_pipeline" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-pipeline"
}
`
}

func TestUnitPipelineDataSource_ReadError(t *testing.T) {
	baseURL, get, _, _, _ := testUnitMockServer(t)
	get.set(testUnitJSON(500, `{"error":"internal error"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitPipelineDataSourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error reading pipeline`),
			},
		},
	})
}
