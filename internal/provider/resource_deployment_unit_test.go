// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitDeploymentAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-deployment","description":"","input_type":"structured","output_type":"structured","supports_request_format":true,"input_fields":[],"output_fields":[],"labels":{},"default_version":null,"creation_date":"2024-01-01T00:00:00.000000Z","last_updated":"2024-01-01T00:00:00.000000Z"}`

func testUnitDeploymentResourceConfig(baseURL, description string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_deployment" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-deployment"
  description  = "` + description + `"
}
`
}

func TestUnitDeploymentResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid deployment name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitDeploymentResourceConfig(baseURL, ""),
				ExpectError: regexp.MustCompile(`Error creating deployment`),
			},
		},
	})
}

func TestUnitDeploymentResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentResourceConfig(baseURL, ""),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading deployment`),
			},
		},
	})
}

func TestUnitDeploymentResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentResourceConfig(baseURL, ""),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitDeploymentResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentResourceConfig(baseURL, ""),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid description"}`)) },
				Config:      testUnitDeploymentResourceConfig(baseURL, "Updated description"),
				ExpectError: regexp.MustCompile(`Error updating deployment`),
			},
		},
	})
}

func TestUnitDeploymentResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete deployment in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentResourceConfig(baseURL, ""),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting deployment`),
			},
		},
	})
}
