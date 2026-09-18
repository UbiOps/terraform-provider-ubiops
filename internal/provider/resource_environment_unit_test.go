// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitEnvironmentAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-environment","base_environment":"python3-12","description":"unit test description","supports_request_format":true,"creation_date":"2024-01-01T00:00:00Z","last_updated":"2024-01-01T00:00:00Z"}`

func testUnitEnvironmentResourceConfig(baseURL, description string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_environment" "test" {
  project_name     = "unit-test-project"
  name             = "unit-test-environment"
  base_environment = "python3-12"
  description      = ` + description + `
}
`
}

func TestUnitEnvironmentResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid environment name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitEnvironmentResourceConfig(baseURL, `"unit test description"`),
				ExpectError: regexp.MustCompile(`Error creating environment`),
			},
		},
	})
}

func TestUnitEnvironmentResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvironmentAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvironmentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvironmentResourceConfig(baseURL, `"unit test description"`),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading environment`),
			},
		},
	})
}

func TestUnitEnvironmentResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvironmentAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvironmentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvironmentResourceConfig(baseURL, `"unit test description"`),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitEnvironmentResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvironmentAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvironmentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvironmentResourceConfig(baseURL, `"unit test description"`),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid description"}`)) },
				Config:      testUnitEnvironmentResourceConfig(baseURL, `"unit test description updated"`),
				ExpectError: regexp.MustCompile(`Error updating environment`),
			},
		},
	})
}

func TestUnitEnvironmentResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitEnvironmentAPIResponse))
	get.set(testUnitJSON(200, testUnitEnvironmentAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete environment in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitEnvironmentResourceConfig(baseURL, `"unit test description"`),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting environment`),
			},
		},
	})
}
