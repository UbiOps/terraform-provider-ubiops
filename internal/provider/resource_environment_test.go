// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"archive/zip"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestReadEnvironmentResult_BaseEnvironment locks in that base_environment
// normalizes to null when absent (e.g. Docker-based custom environments).
func TestReadEnvironmentResult_BaseEnvironment(t *testing.T) {
	tests := []struct {
		name    string
		result  map[string]any
		wantVal string
		wantNil bool
	}{
		{"real value", map[string]any{"base_environment": "python3-12"}, "python3-12", false},
		{"null value", map[string]any{"base_environment": nil}, "", true},
		{"absent key", map[string]any{}, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data EnvironmentResourceModel
			readEnvironmentResult(tt.result, &data)

			if data.BaseEnvironment.IsUnknown() {
				t.Fatal("base_environment must never be left Unknown after Read")
			}
			if tt.wantNil {
				if !data.BaseEnvironment.IsNull() {
					t.Fatalf("got %v, want null", data.BaseEnvironment)
				}
				return
			}
			if data.BaseEnvironment.ValueString() != tt.wantVal {
				t.Fatalf("got %q, want %q", data.BaseEnvironment.ValueString(), tt.wantVal)
			}
		})
	}
}

func TestAccEnvironmentResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	envName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_environment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/environments/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccEnvironmentResourceConfig(projectName, envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("project_name"),
						knownvalue.StringExact(projectName),
					),
				},
			},
			// ImportState.
			{
				ResourceName:            "ubiops_environment.test",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s", projectName, envName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"build_timeout"},
			},
			// Update description.
			{
				Config: testAccEnvironmentResourceConfigUpdated(projectName, envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func TestAccEnvironmentResourceNoBaseEnvironment(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	envName := testAccResourceName(t)

	// Regression test: base_environment omitted used to leave the field
	// Unknown after apply instead of null.
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_environment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/environments/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfigNoBaseEnvironment(projectName, envName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("base_environment"),
						knownvalue.Null(),
					),
				},
			},
			// Re-apply the identical config - must produce an empty plan, not an
			// "inconsistent result" error from base_environment staying Unknown.
			{
				Config:   testAccEnvironmentResourceConfigNoBaseEnvironment(projectName, envName),
				PlanOnly: true,
			},
		},
	})
}

func testAccEnvironmentResourceConfigNoBaseEnvironment(projectName, name string) string {
	return fmt.Sprintf(`
resource "ubiops_environment" "test" {
  project_name = %[1]q
  name         = %[2]q
}
`, projectName, name)
}

func testAccEnvironmentResourceConfig(projectName, name string) string {
	return fmt.Sprintf(`
resource "ubiops_environment" "test" {
  project_name     = %[1]q
  name             = %[2]q
  base_environment = "python3-12"
}
`, projectName, name)
}

func testAccEnvironmentResourceConfigUpdated(projectName, name string) string {
	return fmt.Sprintf(`
resource "ubiops_environment" "test" {
  project_name     = %[1]q
  name             = %[2]q
  base_environment = "python3-12"
  description      = "Updated description"
}
`, projectName, name)
}

func TestAccEnvironmentResource_BuildFromSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	envName := testAccResourceName(t)
	zipV1 := testAccZipRequirementsFile(t, testAccWriteRequirementsFile(t, "v1"))
	zipV2 := testAccZipRequirementsFile(t, testAccWriteRequirementsFile(t, "v2"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_environment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/environments/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			{
				Config: testAccEnvironmentResourceConfigSource(projectName, envName, zipV1, 600),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(envName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("source_file_sha256"),
						knownvalue.NotNull(),
					),
				},
			},
			{
				Config:   testAccEnvironmentResourceConfigSource(projectName, envName, zipV1, 600),
				PlanOnly: true,
			},
			{
				Config: testAccEnvironmentResourceConfigSource(projectName, envName, zipV2, 600),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("source_file_sha256"),
						knownvalue.NotNull(),
					),
				},
			},
		},
	})
}

func testAccEnvironmentResourceConfigSource(projectName, name, sourceFile string, buildTimeout int) string {
	return fmt.Sprintf(`
resource "ubiops_environment" "test" {
  project_name     = %[1]q
  name             = %[2]q
  base_environment = "python3-12"
  source_file      = %[3]q
  build_timeout    = %[4]d
}
`, projectName, name, sourceFile, buildTimeout)
}

func testAccZipRequirementsFile(t *testing.T, reqFile string) string {
	t.Helper()

	zipPath := reqFile + ".zip"
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("requirements.txt")
	if err != nil {
		t.Fatalf("failed to add requirements.txt to zip: %v", err)
	}
	content, err := os.ReadFile(reqFile)
	if err != nil {
		t.Fatalf("failed to read requirements.txt: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("failed to write requirements.txt to zip: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return zipPath
}
