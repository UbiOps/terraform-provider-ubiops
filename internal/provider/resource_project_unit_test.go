// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitProjectAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-project","organization_name":"unit-test-org","advanced_permissions":false,"creation_date":"2024-01-01T00:00:00.000000Z"}`

func testUnitProjectResourceConfig(baseURL, advancedPermissions string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_project" "test" {
  name                 = "unit-test-project"
  organization_name    = "unit-test-org"
  advanced_permissions = ` + advancedPermissions + `
}
`
}

func TestUnitProjectResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid project name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitProjectResourceConfig(baseURL, "false"),
				ExpectError: regexp.MustCompile(`Error creating project`),
			},
		},
	})
}

func TestUnitProjectResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading project`),
			},
		},
	})
}

func TestUnitProjectResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitProjectResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid advanced_permissions"}`)) },
				Config:      testUnitProjectResourceConfig(baseURL, "true"),
				ExpectError: regexp.MustCompile(`Error updating project`),
			},
		},
	})
}

func TestUnitProjectResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete project in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectResourceConfig(baseURL, "false"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting project`),
			},
		},
	})
}

func TestUnitProjectResource_Import(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectResourceConfig(baseURL, "false"),
			},
			{
				ResourceName:      "ubiops_project.test",
				ImportState:       true,
				ImportStateId:     "unit-test-project",
				ImportStateVerify: true,
			},
		},
	})
}
