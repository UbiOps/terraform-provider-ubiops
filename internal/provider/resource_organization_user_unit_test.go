// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitOrganizationUserAPIResponse = `{"id":"22222222-2222-2222-2222-222222222222","email":"unit-test-user@example.com","status":"active","admin":false}`

func testUnitOrganizationUserResourceConfig(baseURL, admin string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_organization_user" "test" {
  organization_name = "unit-test-org"
  email              = "unit-test-user@example.com"
  admin              = ` + admin + `
}
`
}

func TestUnitOrganizationUserResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid email"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitOrganizationUserResourceConfig(baseURL, "false"),
				ExpectError: regexp.MustCompile(`Error creating organization user`),
			},
		},
	})
}

func TestUnitOrganizationUserResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationUserAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationUserResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading organization user`),
			},
		},
	})
}

func TestUnitOrganizationUserResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationUserAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationUserResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitOrganizationUserResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationUserAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationUserResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid admin flag"}`)) },
				Config:      testUnitOrganizationUserResourceConfig(baseURL, "true"),
				ExpectError: regexp.MustCompile(`Error updating organization user`),
			},
		},
	})
}

func TestUnitOrganizationUserResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationUserAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationUserAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete organization user"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationUserResourceConfig(baseURL, "false"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting organization user`),
			},
		},
	})
}

func TestUnitOrganizationUserResource_Import(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationUserAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationUserResourceConfig(baseURL, "false"),
			},
			{
				ResourceName:      "ubiops_organization_user.test",
				ImportState:       true,
				ImportStateId:     "unit-test-org/22222222-2222-2222-2222-222222222222",
				ImportStateVerify: true,
			},
		},
	})
}
