// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitProjectUserAPIResponse = `{"id":"22222222-2222-2222-2222-222222222222","email":"unit-test-user@example.com","name":"Unit","surname":"Tester"}`

func testUnitProjectUserResourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_project_user" "test" {
  project_name = "unit-test-project"
  user_id      = "22222222-2222-2222-2222-222222222222"
}
`
}

func TestUnitProjectUserResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid user_id"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitProjectUserResourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error creating project user`),
			},
		},
	})
}

func TestUnitProjectUserResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectUserAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectUserResourceConfig(baseURL),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading project user`),
			},
		},
	})
}

func TestUnitProjectUserResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectUserAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectUserResourceConfig(baseURL),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitProjectUserResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectUserAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectUserAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete project user"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectUserResourceConfig(baseURL),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting project user`),
			},
		},
	})
}

func TestUnitProjectUserResource_Import(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitProjectUserAPIResponse))
	get.set(testUnitJSON(200, testUnitProjectUserAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitProjectUserResourceConfig(baseURL),
			},
			{
				ResourceName:      "ubiops_project_user.test",
				ImportState:       true,
				ImportStateId:     "unit-test-project/22222222-2222-2222-2222-222222222222",
				ImportStateVerify: true,
			},
		},
	})
}
