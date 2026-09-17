// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccDeploymentResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_deployment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/deployments/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			// Create with structured input/output.
			{
				Config: testAccDeploymentResourceConfig(projectName, deploymentName, "structured"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(deploymentName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("input_type"),
						knownvalue.StringExact("structured"),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("project_name"),
						knownvalue.StringExact(projectName),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_deployment.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectName, deploymentName),
				ImportStateVerify: true,
			},
			// Update description.
			{
				Config: testAccDeploymentResourceConfigUpdated(projectName, deploymentName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccDeploymentResourceConfig(projectName, name, inputType string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name = %[1]q
  name         = %[2]q
  input_type   = %[3]q
  output_type  = %[3]q

  input_fields = [
    {
      name      = "input"
      data_type = "string"
    }
  ]

  output_fields = [
    {
      name      = "output"
      data_type = "string"
    }
  ]
}
`, projectName, name, inputType)
}

func testAccDeploymentResourceConfigUpdated(projectName, name string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name = %[1]q
  name         = %[2]q
  description  = "Updated description"
  input_type   = "structured"
  output_type  = "structured"

  input_fields = [
    {
      name      = "input"
      data_type = "string"
    }
  ]

  output_fields = [
    {
      name      = "output"
      data_type = "string"
    }
  ]
}
`, projectName, name)
}

func TestAccDeploymentResource_InvalidInputType(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name = %[1]q
  name         = %[2]q
  input_type   = "not_a_real_type"
  output_type  = "structured"

  input_fields = [
    { name = "input", data_type = "string" }
  ]

  output_fields = [
    { name = "output", data_type = "string" }
  ]
}
`, projectName, deploymentName),
				ExpectError: regexp.MustCompile(`(?s)Attribute input_type value must be one of`),
			},
		},
	})
}

func TestAccDeploymentResource_DefaultVersionOnCreate(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_deployment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/deployments/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentResourceConfigDefaultVersionOnCreate(projectName, deploymentName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("default_version"),
						knownvalue.StringExact("v1"),
					),
				},
			},
		},
	})
}

func testAccDeploymentResourceConfigDefaultVersionOnCreate(projectName, deploymentName string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name    = %[1]q
  name            = %[2]q
  input_type      = "structured"
  output_type     = "structured"
  default_version = "v1"

  input_fields = [
    {
      name      = "input"
      data_type = "string"
    }
  ]

  output_fields = [
    {
      name      = "output"
      data_type = "string"
    }
  ]
}

resource "ubiops_deployment_version" "test" {
  project_name      = %[1]q
  deployment_name   = %[2]q # literal, not a reference to ubiops_deployment.test.name - avoids a dependency-edge deadlock
  version           = "v1"
  minimum_instances = 0
  maximum_instances = 1
}
`, projectName, deploymentName)
}

func TestAccDeploymentResource_PromoteDefaultVersion(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)
	zipPath := testAccWriteDeploymentPackageZip(t, "promote-v2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_deployment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/deployments/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			{
				Config: testAccDeploymentResourceConfigPromote(projectName, deploymentName, zipPath, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("default_version"),
						knownvalue.StringExact("v1"),
					),
				},
			},
			{
				Config: testAccDeploymentResourceConfigPromote(projectName, deploymentName, zipPath, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("default_version"),
						knownvalue.StringExact("v2"),
					),
				},
			},
		},
	})
}

func testAccDeploymentResourceConfigPromote(projectName, deploymentName, zipPath string, addV2 bool) string {
	extra := ""
	defaultVersion := "v1"
	if addV2 {
		extra = fmt.Sprintf(`
resource "ubiops_deployment_version" "v2" {
  project_name      = %[1]q
  deployment_name   = %[2]q
  version           = "v2"
  environment       = "python3-13"
  source_file       = %[3]q
  build_timeout     = 600
  minimum_instances = 0
  maximum_instances = 1
}
`, projectName, deploymentName, zipPath)
		defaultVersion = "v2"
	}

	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name    = %[1]q
  name            = %[2]q
  input_type      = "structured"
  output_type     = "structured"
  default_version = %[3]q

  input_fields = [
    {
      name      = "input"
      data_type = "string"
    }
  ]

  output_fields = [
    {
      name      = "output"
      data_type = "string"
    }
  ]
}

resource "ubiops_deployment_version" "v1" {
  project_name      = %[1]q
  deployment_name   = %[2]q
  version           = "v1"
  minimum_instances = 0
  maximum_instances = 1
}
%[4]s
`, projectName, deploymentName, defaultVersion, extra)
}
