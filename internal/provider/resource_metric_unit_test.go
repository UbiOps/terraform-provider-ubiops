// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitMetricAPIResponse = `{"id":42,"name":"unit-test-metric","description":"unit-test","metric_type":"gauge","unit":"","custom":true,"creation_date":"2024-01-01T00:00:00Z","last_updated":"2024-01-01T00:00:00Z"}`

func testUnitMetricResourceConfig(baseURL, description string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_metric" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-metric"
  metric_type  = "gauge"
  description  = "` + description + `"
}
`
}

func TestUnitMetricResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid metric name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitMetricResourceConfig(baseURL, "unit-test"),
				ExpectError: regexp.MustCompile(`Error creating metric`),
			},
		},
	})
}

func TestUnitMetricResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitMetricAPIResponse))
	get.set(testUnitJSON(200, testUnitMetricAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitMetricResourceConfig(baseURL, "unit-test"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading metric`),
			},
		},
	})
}

func TestUnitMetricResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitMetricAPIResponse))
	get.set(testUnitJSON(200, testUnitMetricAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitMetricResourceConfig(baseURL, "unit-test"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitMetricResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitMetricAPIResponse))
	get.set(testUnitJSON(200, testUnitMetricAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitMetricResourceConfig(baseURL, "unit-test"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid description"}`)) },
				Config:      testUnitMetricResourceConfig(baseURL, "unit-test-changed"),
				ExpectError: regexp.MustCompile(`Error updating metric`),
			},
		},
	})
}

func TestUnitMetricResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitMetricAPIResponse))
	get.set(testUnitJSON(200, testUnitMetricAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete metric in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitMetricResourceConfig(baseURL, "unit-test"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting metric`),
			},
		},
	})
}
