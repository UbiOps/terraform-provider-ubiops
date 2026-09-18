// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitRoleAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-role","permissions":["deployments.list"]}`

func testUnitRoleResourceConfig(baseURL, permissions string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_role" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-role"
  permissions  = [` + permissions + `]
}
`
}

func TestUnitRoleResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid role name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitRoleResourceConfig(baseURL, `"deployments.list"`),
				ExpectError: regexp.MustCompile(`Error creating role`),
			},
		},
	})
}

func TestUnitRoleResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRoleAPIResponse))
	get.set(testUnitJSON(200, testUnitRoleAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRoleResourceConfig(baseURL, `"deployments.list"`),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading role`),
			},
		},
	})
}

func TestUnitRoleResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRoleAPIResponse))
	get.set(testUnitJSON(200, testUnitRoleAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRoleResourceConfig(baseURL, `"deployments.list"`),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitRoleResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRoleAPIResponse))
	get.set(testUnitJSON(200, testUnitRoleAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRoleResourceConfig(baseURL, `"deployments.list"`),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid permissions"}`)) },
				Config:      testUnitRoleResourceConfig(baseURL, `"deployments.get"`),
				ExpectError: regexp.MustCompile(`Error updating role`),
			},
		},
	})
}

func TestUnitRoleResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRoleAPIResponse))
	get.set(testUnitJSON(200, testUnitRoleAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete role in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRoleResourceConfig(baseURL, `"deployments.list"`),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting role`),
			},
		},
	})
}
