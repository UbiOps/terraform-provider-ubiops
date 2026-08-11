// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

func TestAccWebhookResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t) + "-dep"
	webhookName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create deployment + version first - default_version can't reference a version
			// that doesn't exist yet, so it's set in a later step once "v1" is created.
			{
				Config: testAccWebhookDeploymentOnlyConfig(projectName, deploymentName),
			},
			// Create webhook; promotes "v1" to default_version now that it exists.
			{
				Config: testAccWebhookResourceConfig(projectName, deploymentName, webhookName, "https://example.com/hook"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_webhook.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(webhookName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_webhook.test",
						tfjsonpath.New("event"),
						knownvalue.StringExact("deployment_request_finished"),
					),
				},
			},
			// ImportState.
			{
				ResourceName:            "ubiops_webhook.test",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s", projectName, webhookName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"headers_json"},
			},
			// Update URL.
			{
				Config: testAccWebhookResourceConfig(projectName, deploymentName, webhookName, "https://example.com/hook-updated"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_webhook.test",
						tfjsonpath.New("url"),
						knownvalue.StringExact("https://example.com/hook-updated"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccWebhookDeploymentOnlyConfig(projectName, deploymentName string) string {
	return fmt.Sprintf(`
resource "ubiops_deployment" "test" {
  project_name = %[1]q
  name         = %[2]q
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

resource "ubiops_deployment_version" "test" {
  project_name      = %[1]q
  deployment_name   = ubiops_deployment.test.name
  version           = "v1"
  minimum_instances = 0
  maximum_instances = 1
}
`, projectName, deploymentName)
}

func testAccWebhookResourceConfig(projectName, deploymentName, webhookName, url string) string {
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
  deployment_name   = ubiops_deployment.test.name
  version           = "v1"
  minimum_instances = 0
  maximum_instances = 1
}

resource "ubiops_webhook" "test" {
  project_name = %[1]q
  name         = %[3]q
  url          = %[4]q
  event        = "deployment_request_finished"
  object_type  = "deployment"
  object_name  = ubiops_deployment.test.name

  depends_on = [ubiops_deployment_version.test]
}
`, projectName, deploymentName, webhookName, url)
}
