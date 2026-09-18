// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitRequestScheduleAPIResponse = `{"id":"22222222-2222-2222-2222-222222222222","name":"unit-test-schedule","description":"","object_type":"deployment","object_name":"unit-test-deployment","schedule":"0 0 * * *","enabled":true,"timeout":14400,"creation_date":"2024-01-01T00:00:00Z"}`

func testUnitRequestScheduleResourceConfig(baseURL, schedule string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_request_schedule" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-schedule"
  object_type  = "deployment"
  object_name  = "unit-test-deployment"
  schedule     = ` + schedule + `
}
`
}

func TestUnitRequestScheduleResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid schedule"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitRequestScheduleResourceConfig(baseURL, `"0 0 * * *"`),
				ExpectError: regexp.MustCompile(`Error creating request schedule`),
			},
		},
	})
}

func TestUnitRequestScheduleResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRequestScheduleAPIResponse))
	get.set(testUnitJSON(200, testUnitRequestScheduleAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRequestScheduleResourceConfig(baseURL, `"0 0 * * *"`),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading request schedule`),
			},
		},
	})
}

func TestUnitRequestScheduleResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRequestScheduleAPIResponse))
	get.set(testUnitJSON(200, testUnitRequestScheduleAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRequestScheduleResourceConfig(baseURL, `"0 0 * * *"`),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitRequestScheduleResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRequestScheduleAPIResponse))
	get.set(testUnitJSON(200, testUnitRequestScheduleAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRequestScheduleResourceConfig(baseURL, `"0 0 * * *"`),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid schedule"}`)) },
				Config:      testUnitRequestScheduleResourceConfig(baseURL, `"0 12 * * *"`),
				ExpectError: regexp.MustCompile(`Error updating request schedule`),
			},
		},
	})
}

func TestUnitRequestScheduleResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitRequestScheduleAPIResponse))
	get.set(testUnitJSON(200, testUnitRequestScheduleAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete schedule in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitRequestScheduleResourceConfig(baseURL, `"0 0 * * *"`),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting request schedule`),
			},
		},
	})
}
