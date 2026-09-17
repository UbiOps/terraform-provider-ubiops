// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitWebhookAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-webhook","description":"","url":"https://example.com/hook","event":"deployment_request_finished","object_type":"deployment","object_name":"unit-test-deployment","enabled":true,"retry":false,"include_result":false,"timeout":10,"creation_date":"2024-01-01T00:00:00Z","last_updated":"2024-01-01T00:00:00Z"}`

func testUnitWebhookResourceConfig(baseURL, url string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_webhook" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-webhook"
  url          = "` + url + `"
  event        = "deployment_request_finished"
  object_type  = "deployment"
  object_name  = "unit-test-deployment"
}
`
}

func TestUnitWebhookResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid webhook url"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitWebhookResourceConfig(baseURL, "https://example.com/hook"),
				ExpectError: regexp.MustCompile(`Error creating webhook`),
			},
		},
	})
}

func TestUnitWebhookResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitWebhookAPIResponse))
	get.set(testUnitJSON(200, testUnitWebhookAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitWebhookResourceConfig(baseURL, "https://example.com/hook"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading webhook`),
			},
		},
	})
}

func TestUnitWebhookResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitWebhookAPIResponse))
	get.set(testUnitJSON(200, testUnitWebhookAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitWebhookResourceConfig(baseURL, "https://example.com/hook"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitWebhookResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitWebhookAPIResponse))
	get.set(testUnitJSON(200, testUnitWebhookAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitWebhookResourceConfig(baseURL, "https://example.com/hook"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid url"}`)) },
				Config:      testUnitWebhookResourceConfig(baseURL, "https://example.com/hook-changed"),
				ExpectError: regexp.MustCompile(`Error updating webhook`),
			},
		},
	})
}

func TestUnitWebhookResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitWebhookAPIResponse))
	get.set(testUnitJSON(200, testUnitWebhookAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete webhook in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitWebhookResourceConfig(baseURL, "https://example.com/hook"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting webhook`),
			},
		},
	})
}
