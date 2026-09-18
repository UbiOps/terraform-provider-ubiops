// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitPipelineVersionAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","version":"v1","description":"","request_retention_mode":"full","request_retention_time":604800,"creation_date":"2024-01-01T00:00:00Z","last_updated":"2024-01-01T00:00:00Z"}`

func testUnitPipelineVersionResourceConfig(baseURL, version string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_pipeline_version" "test" {
  project_name  = "unit-test-project"
  pipeline_name = "unit-test-pipeline"
  version       = "` + version + `"
}
`
}

func TestUnitPipelineVersionResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid pipeline version"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitPipelineVersionResourceConfig(baseURL, "v1"),
				ExpectError: regexp.MustCompile(`Error creating pipeline version`),
			},
		},
	})
}

func TestUnitPipelineVersionResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineVersionAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineVersionResourceConfig(baseURL, "v1"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading pipeline version`),
			},
		},
	})
}

func TestUnitPipelineVersionResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineVersionAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineVersionResourceConfig(baseURL, "v1"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitPipelineVersionResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineVersionAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineVersionResourceConfig(baseURL, "v1"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid pipeline version update"}`)) },
				Config:      testUnitPipelineVersionResourceConfig(baseURL, "v2"),
				ExpectError: regexp.MustCompile(`Error updating pipeline version`),
			},
		},
	})
}

func TestUnitPipelineVersionResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitPipelineVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitPipelineVersionAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete pipeline version in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitPipelineVersionResourceConfig(baseURL, "v1"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting pipeline version`),
			},
		},
	})
}
