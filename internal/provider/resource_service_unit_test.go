// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitServiceAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-service","description":"","deployment":"unit-test-deployment","port":8080,"authentication_required":true,"authentication_method_token_enabled":true,"time_created":"2024-01-01T00:00:00Z","time_updated":"2024-01-01T00:00:00Z"}`

func testUnitServiceResourceConfig(baseURL, description string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_service" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-service"
  deployment   = "unit-test-deployment"
  port         = 8080
  description  = "` + description + `"
}
`
}

func TestUnitServiceResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid service configuration"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitServiceResourceConfig(baseURL, ""),
				ExpectError: regexp.MustCompile(`Error creating service`),
			},
		},
	})
}

func TestUnitServiceResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceResourceConfig(baseURL, ""),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading service`),
			},
		},
	})
}

func TestUnitServiceResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceResourceConfig(baseURL, ""),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitServiceResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceResourceConfig(baseURL, ""),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid update"}`)) },
				Config:      testUnitServiceResourceConfig(baseURL, "updated description"),
				ExpectError: regexp.MustCompile(`Error updating service`),
			},
		},
	})
}

func TestUnitServiceResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitServiceAPIResponse))
	get.set(testUnitJSON(200, testUnitServiceAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete service in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitServiceResourceConfig(baseURL, ""),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting service`),
			},
		},
	})
}
