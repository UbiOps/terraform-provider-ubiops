// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitOrganizationAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-org","status":"active","creation_date":"2024-01-01T00:00:00Z","subscription":"test-plan","two_factor_authentication_forced":false}`

func testUnitOrganizationResourceConfig(baseURL, tfaForced string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_organization" "test" {
  name                              = "unit-test-org"
  two_factor_authentication_forced = ` + tfaForced + `
}
`
}

func TestUnitOrganizationResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid organization name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitOrganizationResourceConfig(baseURL, "false"),
				ExpectError: regexp.MustCompile(`Error creating organization`),
			},
		},
	})
}

func TestUnitOrganizationResource_ReadError(t *testing.T) {
	baseURL, get, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationAPIResponse))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading organization`),
			},
		},
	})
}

func TestUnitOrganizationResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationAPIResponse))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitOrganizationResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, _ := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationAPIResponse))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationResourceConfig(baseURL, "false"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid subscription"}`)) },
				Config:      testUnitOrganizationResourceConfig(baseURL, "true"),
				ExpectError: regexp.MustCompile(`Error updating organization`),
			},
		},
	})
}

func TestUnitOrganizationResource_Import(t *testing.T) {
	baseURL, get, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitOrganizationAPIResponse))
	get.set(testUnitJSON(200, testUnitOrganizationAPIResponse))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitOrganizationResourceConfig(baseURL, "false"),
			},
			{
				ResourceName:      "ubiops_organization.test",
				ImportState:       true,
				ImportStateId:     "unit-test-org",
				ImportStateVerify: true,
			},
		},
	})
}
