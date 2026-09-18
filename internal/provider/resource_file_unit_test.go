// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

const testUnitFileAPIResponse = `{"file":"unit-test-file.txt","size":0,"time_created":"2024-01-01T00:00:00Z"}`

func testUnitFileResourceConfig(baseURL string) string {
	return testUnitProviderConfig(baseURL) + `
resource "ubiops_file" "test" {
  project_name = "unit-test-project"
  bucket_name  = "unit-test-bucket"
  file         = "unit-test-file.txt"
}
`
}

func TestUnitFileResource_CreateError(t *testing.T) {
	baseURL, _, post, _, _ := testUnitMockServer(t)
	post.set(testUnitJSON(400, `{"error":"invalid file"}`))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testUnitFileResourceConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error creating file`),
			},
		},
	})
}

func TestUnitFileResource_ReadError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, `{}`))
	get.set(testUnitJSON(200, testUnitFileAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitFileResourceConfig(baseURL),
			},
			{
				PreConfig:    func() { get.set(testUnitJSON(400, `{"error":"internal error"}`)) },
				RefreshState: true,
				ExpectError:  regexp.MustCompile(`Error reading file`),
			},
		},
	})
}

func TestUnitFileResource_ReadNotFoundDrift(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, `{}`))
	get.set(testUnitJSON(200, testUnitFileAPIResponse))
	del.set(testUnitNoContent())

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitFileResourceConfig(baseURL),
			},
			{
				PreConfig:          func() { get.set(testUnitJSON(404, `{"error":"not found"}`)) },
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestUnitFileResource_DeleteError(t *testing.T) {
	baseURL, get, post, _, del := testUnitMockServer(t)
	post.set(testUnitJSON(201, `{}`))
	get.set(testUnitJSON(200, testUnitFileAPIResponse))
	del.set(testUnitOnceThen(testUnitJSON(400, `{"error":"cannot delete file"}`), testUnitNoContent()))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testUnitFileResourceConfig(baseURL),
			},
			{
				Config:      testUnitProviderConfig(baseURL),
				ExpectError: regexp.MustCompile(`Error deleting file`),
			},
		},
	})
}
