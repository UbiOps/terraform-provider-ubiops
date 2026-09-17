// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitRoleAssignmentAPIResponse = `{"id":"22222222-2222-2222-2222-222222222222","role":"project-viewer","assignee":"11111111-1111-1111-1111-111111111111","assignee_type":"user"}`

func testUnitRoleAssignmentResourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_role_assignment" "test" {
  project_name  = "unit-test-project"
  role          = "project-viewer"
  assignee      = "11111111-1111-1111-1111-111111111111"
  assignee_type = "user"
}
`
}

func TestUnitRoleAssignmentResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid assignee"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitRoleAssignmentResourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error creating role assignment`),
			},
		},
	})
}

func TestUnitRoleAssignmentResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRoleAssignmentAPIResponse))
	get.set(testUnitJSON(200, testUnitRoleAssignmentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRoleAssignmentResourceConfig(baseURL),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading role assignment`),
			},
		},
	})
}

func TestUnitRoleAssignmentResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRoleAssignmentAPIResponse))
	get.set(testUnitJSON(200, testUnitRoleAssignmentAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRoleAssignmentResourceConfig(baseURL),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitRoleAssignmentResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRoleAssignmentAPIResponse))
	get.set(testUnitJSON(200, testUnitRoleAssignmentAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete role assignment"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRoleAssignmentResourceConfig(baseURL),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting role assignment`),
			},
		},
	})
}
