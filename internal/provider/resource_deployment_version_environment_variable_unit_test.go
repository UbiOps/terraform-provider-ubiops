// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitEnvVarAPIResponse = `{"id":"22222222-2222-2222-2222-222222222222","name":"unit-test-var","secret":false}`

func testUnitEnvVarResourceConfig(baseURL, value string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_deployment_version_environment_variable" "test" {
  project_name    = "unit-test-project"
  deployment_name = "unit-test-deployment"
  version         = "v1"
  name            = "unit-test-var"
  value           = "` + value + `"
}
`
}

func TestUnitDeploymentVersionEnvironmentVariableResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid environment variable"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitEnvVarResourceConfig(baseURL, "unit-test-value"),
				ExpectError: regexp.MustCompile(`Error creating environment variable`),
			},
		},
	})
}

func TestUnitDeploymentVersionEnvironmentVariableResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvVarAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvVarResourceConfig(baseURL, "unit-test-value"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading environment variable`),
			},
		},
	})
}

func TestUnitDeploymentVersionEnvironmentVariableResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvVarAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvVarResourceConfig(baseURL, "unit-test-value"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitDeploymentVersionEnvironmentVariableResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvVarAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvVarResourceConfig(baseURL, "unit-test-value"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid value"}`)) },
				Config:      testUnitEnvVarResourceConfig(baseURL, "unit-test-value-2"),
				ExpectError: regexp.MustCompile(`Error updating environment variable`),
			},
		},
	})
}

func TestUnitDeploymentVersionEnvironmentVariableResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvVarAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete environment variable"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvVarResourceConfig(baseURL, "unit-test-value"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting environment variable`),
			},
		},
	})
}
