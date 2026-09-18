// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitDeploymentVersionAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","version":"unit-test-version","description":"unit-test-description","environment":"python3-11","instance_type_group_name":"256mb","request_retention_mode":"full","request_retention_time":604800,"scaling_strategy":"default","minimum_instances":0,"maximum_instances":5,"maximum_idle_time":300,"instance_processes":1,"static_ip":false,"restart_request_interruption":false,"creation_date":"2024-01-01T00:00:00Z","last_updated":"2024-01-01T00:00:00Z"}`

func testUnitDeploymentVersionResourceConfig(baseURL, description string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_deployment_version" "test" {
  project_name    = "unit-test-project"
  deployment_name = "unit-test-deployment"
  version         = "unit-test-version"
  description     = "` + description + `"
}
`
}

func TestUnitDeploymentVersionResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid version"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitDeploymentVersionResourceConfig(baseURL, "unit-test-description"),
				ExpectError: regexp.MustCompile(`Error creating deployment version`),
			},
		},
	})
}

func TestUnitDeploymentVersionResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentVersionAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentVersionResourceConfig(baseURL, "unit-test-description"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading deployment version`),
			},
		},
	})
}

func TestUnitDeploymentVersionResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentVersionAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentVersionResourceConfig(baseURL, "unit-test-description"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitDeploymentVersionResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentVersionAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentVersionResourceConfig(baseURL, "unit-test-description"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid description"}`)) },
				Config:      testUnitDeploymentVersionResourceConfig(baseURL, "updated-description"),
				ExpectError: regexp.MustCompile(`Error updating deployment version`),
			},
		},
	})
}

func TestUnitDeploymentVersionResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentVersionAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentVersionAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete version in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentVersionResourceConfig(baseURL, "unit-test-description"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting deployment version`),
			},
		},
	})
}
