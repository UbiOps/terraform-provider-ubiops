// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitBucketAPIResponse = `{"id":"11111111-1111-1111-1111-111111111111","name":"unit-test-bucket","description":"unit-test","provider":"ubiops","creation_date":"2024-01-01T00:00:00Z","last_updated":"2024-01-01T00:00:00Z"}`

func testUnitBucketResourceConfig(baseURL, description string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_bucket" "test" {
  project_name = "unit-test-project"
  name         = "unit-test-bucket"
  description  = "` + description + `"
}
`
}

func TestUnitBucketResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid bucket name"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitBucketResourceConfig(baseURL, "unit-test"),
				ExpectError: regexp.MustCompile(`Error creating bucket`),
			},
		},
	})
}

func TestUnitBucketResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitBucketAPIResponse))
	get.set(testUnitJSON(200, testUnitBucketAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitBucketResourceConfig(baseURL, "unit-test"),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading bucket`),
			},
		},
	})
}

func TestUnitBucketResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitBucketAPIResponse))
	get.set(testUnitJSON(200, testUnitBucketAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitBucketResourceConfig(baseURL, "unit-test"),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitBucketResource_UpdateError(t *testing.T) {
	baseURL, get, post, patch, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitBucketAPIResponse))
	get.set(testUnitJSON(200, testUnitBucketAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitBucketResourceConfig(baseURL, "unit-test"),
			},
			{
				PreConfig:   func() { patch.set(testUnitJSON(400, `{"error":"invalid description"}`)) },
				Config:      testUnitBucketResourceConfig(baseURL, "unit-test-changed"),
				ExpectError: regexp.MustCompile(`Error updating bucket`),
			},
		},
	})
}

func TestUnitBucketResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, testUnitBucketAPIResponse))
	get.set(testUnitJSON(200, testUnitBucketAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete bucket in use"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitBucketResourceConfig(baseURL, "unit-test"),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting bucket`),
			},
		},
	})
}
