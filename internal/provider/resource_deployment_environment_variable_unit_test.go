// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitDeploymentEnvVarAPIResponse = `{"id":"44444444-4444-4444-4444-444444444444","name":"unit-test-var","secret":false}`

func testUnitDeploymentEnvVarResourceConfig(baseURL, value string) string {
	return testUnitProviderConfig(baseURL) + fmt.Sprintf(`
resource "ubiops_deployment_environment_variable" "test" {
  project_name    = "unit-test-project"
  deployment_name = "unit-test-deployment"
  name            = "unit-test-var"
  value           = %q
}
`, value)
}

func TestUnitDeploymentEnvironmentVariableResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid variable name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitDeploymentEnvVarResourceConfig(baseURL, "initial-value"),
				ExpectError: regexp.MustCompile(`Error creating environment variable`),
			},
		},
	})
}

func TestUnitDeploymentEnvironmentVariableResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentEnvVarAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentEnvVarResourceConfig(baseURL, "initial-value"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading environment variable`),
			},
		},
	})
}

func TestUnitDeploymentEnvironmentVariableResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentEnvVarAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentEnvVarResourceConfig(baseURL, "initial-value"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitDeploymentEnvironmentVariableResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentEnvVarAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentEnvVarResourceConfig(baseURL, "initial-value"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid value"}`)) },
				Config:      testUnitDeploymentEnvVarResourceConfig(baseURL, "updated-value"),
				ExpectError: regexp.MustCompile(`Error updating environment variable`),
			},
		},
	})
}

func TestUnitDeploymentEnvironmentVariableResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitDeploymentEnvVarAPIResponse))
	get.set(testUnitJSON(200, testUnitDeploymentEnvVarAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete environment variable"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitDeploymentEnvVarResourceConfig(baseURL, "initial-value"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting environment variable`),
			},
		},
	})
}
