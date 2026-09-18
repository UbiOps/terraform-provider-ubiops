// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitInstanceTypeGroupAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-group","time_created":"2024-01-01T00:00:00Z","time_updated":"2024-01-01T00:00:00Z","instance_types":[{"id":"unit-test-instance-type","priority":0}]}`

func testUnitInstanceTypeGroupResourceConfig(baseURL, name string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_instance_type_group" "test" {
  project_name = "unit-test-project"
  name         = ` + name + `

  instance_types_json = jsonencode([
    {
      id       = "unit-test-instance-type"
      priority = 0
    }
  ])
}
`
}

func TestUnitInstanceTypeGroupResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid instance type group name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitInstanceTypeGroupResourceConfig(baseURL, `"unit-test-group"`),
				ExpectError: regexp.MustCompile(`Error creating instance type group`),
			},
		},
	})
}

func TestUnitInstanceTypeGroupResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitInstanceTypeGroupAPIResponse))
	get.set(testUnitJSON(200, testUnitInstanceTypeGroupAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitInstanceTypeGroupResourceConfig(baseURL, `"unit-test-group"`),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading instance type group`),
			},
		},
	})
}

func TestUnitInstanceTypeGroupResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitInstanceTypeGroupAPIResponse))
	get.set(testUnitJSON(200, testUnitInstanceTypeGroupAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitInstanceTypeGroupResourceConfig(baseURL, `"unit-test-group"`),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitInstanceTypeGroupResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitInstanceTypeGroupAPIResponse))
	get.set(testUnitJSON(200, testUnitInstanceTypeGroupAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitInstanceTypeGroupResourceConfig(baseURL, `"unit-test-group"`),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid instance types"}`)) },
				Config:      testUnitInstanceTypeGroupResourceConfig(baseURL, `"unit-test-group-renamed"`),
				ExpectError: regexp.MustCompile(`Error updating instance type group`),
			},
		},
	})
}

func TestUnitInstanceTypeGroupResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitInstanceTypeGroupAPIResponse))
	get.set(testUnitJSON(200, testUnitInstanceTypeGroupAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete instance type group in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitInstanceTypeGroupResourceConfig(baseURL, `"unit-test-group"`),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting instance type group`),
			},
		},
	})
}
