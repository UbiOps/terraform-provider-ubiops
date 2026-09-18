// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitServiceUserAPIResponse = `{"id":"22222222-2222-2222-2222-222222222222","name":"unit-test-user","description":"unit-test description","email":"unit-test-user@example.com","creation_date":"2024-01-01T00:00:00Z"}`

func testUnitServiceUserResourceConfig(baseURL, description string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_service_user" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-user"
  description  = "` + description + `"
}
`
}

func TestUnitServiceUserResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid service user configuration"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitServiceUserResourceConfig(baseURL, "unit-test description"),
				ExpectError: regexp.MustCompile(`Error creating service user`),
			},
		},
	})
}

func TestUnitServiceUserResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceUserAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceUserResourceConfig(baseURL, "unit-test description"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading service user`),
			},
		},
	})
}

func TestUnitServiceUserResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceUserAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceUserResourceConfig(baseURL, "unit-test description"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitServiceUserResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceUserAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceUserResourceConfig(baseURL, "unit-test description"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid update"}`)) },
				Config:      testUnitServiceUserResourceConfig(baseURL, "updated description"),
				ExpectError: regexp.MustCompile(`Error updating service user`),
			},
		},
	})
}

func TestUnitServiceUserResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceUserAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceUserAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete service user in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceUserResourceConfig(baseURL, "unit-test description"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting service user`),
			},
		},
	})
}
