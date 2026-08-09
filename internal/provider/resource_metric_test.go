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

func TestAccMetricResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	metricName := "custom." + testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccMetricResourceConfig(projectName, metricName, "gauge", ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_metric.test",
						tfjsonpath.New("name"),
						knownvalue.StringExact(metricName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_metric.test",
						tfjsonpath.New("metric_type"),
						knownvalue.StringExact("gauge"),
					),
					statecheck.ExpectKnownValue(
						"ubiops_metric.test",
						tfjsonpath.New("custom"),
						knownvalue.Bool(true),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_metric.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s", projectName, metricName),
				ImportStateVerify: true,
			},
			// Update description.
			{
				Config: testAccMetricResourceConfig(projectName, metricName, "gauge", "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_metric.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccMetricResourceConfig(projectName, name, metricType, description string) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("\n  description = %q", description)
	}
	return fmt.Sprintf(`
resource "ubiops_metric" "test" {
  project_name = %[1]q
  name         = %[2]q
  metric_type  = %[3]q%[4]s
}
`, projectName, name, metricType, desc)
}
