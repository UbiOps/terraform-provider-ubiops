// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitPipelineAPIResponse = `{"id":"44444444-4444-4444-4444-444444444444","name":"unit-test-pipeline","description":"","input_type":"structured","output_type":"structured","creation_date":"2024-01-01T00:00:00Z","last_updated":"2024-01-01T00:00:00Z"}`

func testUnitPipelineResourceConfig(baseURL, extra string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_pipeline" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-pipeline"
  input_type   = "structured"
` + extra + `
}
`
}

func TestUnitPipelineResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid pipeline name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitPipelineResourceConfig(baseURL, ""),
				ExpectError: regexp.MustCompile(`Error creating pipeline`),
			},
		},
	})
}

func TestUnitPipelineResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineResourceConfig(baseURL, ""),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading pipeline`),
			},
		},
	})
}

func TestUnitPipelineResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineResourceConfig(baseURL, ""),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitPipelineResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineResourceConfig(baseURL, ""),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid description"}`)) },
				Config:      testUnitPipelineResourceConfig(baseURL, `  description = "unit-test-updated"`),
				ExpectError: regexp.MustCompile(`Error updating pipeline`),
			},
		},
	})
}

func TestUnitPipelineResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete pipeline in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineResourceConfig(baseURL, ""),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting pipeline`),
			},
		},
	})
}
