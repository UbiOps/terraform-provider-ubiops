// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
)

// TestOrderByRef and TestPipelineObjectsFromAPI_PreservesOrder test
// reordering directly, without a live API.
func TestOrderByRef(t *testing.T) {
	tests := []struct {
		name     string
		apiOrder []string
		refOrder []string
		want     []string
	}{
		{"matches ref order", []string{"a", "b"}, []string{"b", "a"}, []string{"b", "a"}},
		{"no ref order falls back to api order", []string{"a", "b"}, nil, []string{"a", "b"}},
		{"new item appended after known items", []string{"a", "b", "c"}, []string{"b", "a"}, []string{"b", "a", "c"}},
		{"removed item is dropped", []string{"a"}, []string{"b", "a"}, []string{"a"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := orderByRef(tt.apiOrder, tt.refOrder)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestPipelineObjectsFromAPI_PreservesOrder(t *testing.T) {
	// API returns objects alphabetically; ref order has "b" before "a" - Read must preserve "b", "a".
	ctx := context.Background()
	apiResponse := []any{
		map[string]any{"name": "a", "reference_type": "deployment", "reference_name": "dep-a"},
		map[string]any{"name": "b", "reference_type": "deployment", "reference_name": "dep-b"},
	}

	refObj := func(name string) attr.Value {
		v, _ := types.ObjectValue(pipelineObjectType.AttrTypes, map[string]attr.Value{
			"name":               types.StringValue(name),
			"reference_type":     types.StringValue("deployment"),
			"reference_name":     types.StringValue("dep-" + name),
			"version":            types.StringNull(),
			"configuration_json": types.StringNull(),
		})
		return v
	}
	ref, _ := types.ListValue(pipelineObjectType, []attr.Value{refObj("b"), refObj("a")})

	got := pipelineObjectsFromAPI(apiResponse, pipelineObjectRefOrder(ctx, ref))
	if got.IsNull() {
		t.Fatal("expected a non-null list")
	}

	elems := got.Elements()
	if len(elems) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(elems))
	}
	wantOrder := []string{"b", "a"}
	for i, elem := range elems {
		obj, ok := elem.(types.Object)
		if !ok {
			t.Fatalf("element %d is not an object: %T", i, elem)
		}
		nameVal, ok := obj.Attributes()["name"].(types.String)
		if !ok {
			t.Fatalf("element %d has no name attribute", i)
		}
		if nameVal.ValueString() != wantOrder[i] {
			t.Fatalf("element %d: got name %q, want %q", i, nameVal.ValueString(), wantOrder[i])
		}
	}
}

func TestAccPipelineVersionResource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	pipelineName := testAccResourceName(t)
	versionName := "v1"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_pipeline_version", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/pipelines/%s/versions/%s", a["project_name"], a["pipeline_name"], a["version"])
		}),
		Steps: []resource.TestStep{
			// Create.
			{
				Config: testAccPipelineVersionResourceConfig(projectName, pipelineName, versionName, ""),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact(versionName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("request_retention_mode"),
						knownvalue.StringExact("full"),
					),
				},
			},
			// ImportState.
			{
				ResourceName:      "ubiops_pipeline_version.test",
				ImportState:       true,
				ImportStateId:     fmt.Sprintf("%s/%s/%s", projectName, pipelineName, versionName),
				ImportStateVerify: true,
			},
			// Update description.
			{
				Config: testAccPipelineVersionResourceConfig(projectName, pipelineName, versionName, "Updated description"),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("description"),
						knownvalue.StringExact("Updated description"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func TestAccPipelineVersionResourceObjectOrdering(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	pipelineName := testAccResourceName(t)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_pipeline_version", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/pipelines/%s/versions/%s", a["project_name"], a["pipeline_name"], a["version"])
		}),
		Steps: []resource.TestStep{
			// Objects declared out of alphabetical order - regression test for
			// state not matching config's exact order.
			{
				Config: testAccPipelineVersionObjectOrderingConfig(projectName, pipelineName),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("objects").AtSliceIndex(0).AtMapKey("name"),
						knownvalue.StringExact("step-b"),
					),
					statecheck.ExpectKnownValue(
						"ubiops_pipeline_version.test",
						tfjsonpath.New("objects").AtSliceIndex(1).AtMapKey("name"),
						knownvalue.StringExact("step-a"),
					),
				},
			},
			// Cold import has no prior order to match - falls back to alphabetical.
			{
				ResourceName:            "ubiops_pipeline_version.test",
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("%s/%s/v1", projectName, pipelineName),
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"objects", "attachments"},
			},
		},
	})
}

func testAccPipelineVersionObjectOrderingConfig(projectName, pipelineName string) string {
	return fmt.Sprintf(`
resource "ubiops_pipeline" "test" {
  project_name = %[1]q
  name         = %[2]q
  input_type   = "structured"
  output_type  = "structured"

  input_fields = [
    { name = "input", data_type = "string" }
  ]

  output_fields = [
    { name = "output", data_type = "string" }
  ]
}

resource "ubiops_pipeline_version" "test" {
  project_name  = %[1]q
  pipeline_name = ubiops_pipeline.test.name
  version       = "v1"

  objects = [
    {
      name           = "step-b"
      reference_type = "operator"
      reference_name = "function"
      configuration_json = jsonencode({
        expression    = "output = input"
        input_fields  = [{ name = "input", data_type = "string" }]
        output_fields = [{ name = "output", data_type = "string" }]
      })
    },
    {
      name           = "step-a"
      reference_type = "operator"
      reference_name = "function"
      configuration_json = jsonencode({
        expression    = "output = input"
        input_fields  = [{ name = "input", data_type = "string" }]
        output_fields = [{ name = "output", data_type = "string" }]
      })
    }
  ]

  attachments = [
    {
      destination_name = "step-b"
      sources = [
        {
          source_name = "pipeline_start"
          mapping = [
            { source_field_name = "input", destination_field_name = "input" }
          ]
        }
      ]
    },
    {
      destination_name = "step-a"
      sources = [
        {
          source_name = "step-b"
          mapping = [
            { source_field_name = "output", destination_field_name = "input" }
          ]
        }
      ]
    },
    {
      destination_name = "pipeline_end"
      sources = [
        {
          source_name = "step-a"
          mapping = [
            { source_field_name = "output", destination_field_name = "output" }
          ]
        }
      ]
    }
  ]
}
`, projectName, pipelineName)
}

func testAccPipelineVersionResourceConfig(projectName, pipelineName, version, description string) string {
	desc := ""
	if description != "" {
		desc = fmt.Sprintf("\n  description = %q", description)
	}
	return fmt.Sprintf(`
resource "ubiops_pipeline" "test" {
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

resource "ubiops_pipeline_version" "test" {
  project_name = %[1]q
  pipeline_name = ubiops_pipeline.test.name
  version       = %[3]q%[4]s
}
`, projectName, pipelineName, version, desc)
}
