// Copyright (c) Dutch Analytics B.V. 2026
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"terraform-provider-ubiops/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// testAccResourceName builds a lowercase, unique, UbiOps-safe resource name,
// truncated to fit the 64-char limit alongside caller-added suffixes.
func testAccResourceName(t *testing.T) string {
	t.Helper()
	name := strings.ToLower(strings.TrimPrefix(t.Name(), "TestAcc"))
	// Some resource types reject anything outside a-z0-9-; strip separators.
	name = nonAlphanumericRun.ReplaceAllString(name, "")
	if len(name) > 20 {
		name = name[:20]
	}
	suffix := strings.ToLower(rand.Text()[:8])
	return fmt.Sprintf("tf-acc-%s-%s", name, suffix)
}

var nonAlphanumericRun = regexp.MustCompile(`[^a-z0-9]+`)

// testAccFirstInstanceTypeID looks up a real instance type UUID, required
// since ubiops_instance_type_group rejects an empty instance_types list.
func testAccFirstInstanceTypeID(t *testing.T, projectName string) string {
	t.Helper()

	baseURL := os.Getenv("UBIOPS_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.ubiops.com/v2.1"
	}
	c := client.NewUbiOpsClient(baseURL, os.Getenv("UBIOPS_API_TOKEN"), "test", "")

	var page struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
	}
	if err := c.Get(context.Background(), fmt.Sprintf("/projects/%s/instance-types", projectName), &page); err != nil {
		t.Fatalf("failed to list instance types: %v", err)
	}
	if len(page.Results) == 0 {
		t.Fatal("project has no available instance types")
	}
	return page.Results[0].ID
}

// testAccCheckDestroyed builds a CheckDestroy func: 404s pathFn(attrs) for every resourceType instance.
func testAccCheckDestroyed(resourceType string, pathFn func(attrs map[string]string) string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		baseURL := os.Getenv("UBIOPS_BASE_URL")
		if baseURL == "" {
			baseURL = "https://api.ubiops.com/v2.1"
		}
		c := client.NewUbiOpsClient(baseURL, os.Getenv("UBIOPS_API_TOKEN"), "test", "")

		for _, rs := range s.RootModule().Resources {
			if rs.Type != resourceType {
				continue
			}
			var result map[string]any
			err := c.Get(context.Background(), pathFn(rs.Primary.Attributes), &result)
			if err == nil {
				return fmt.Errorf("%s %s still exists", resourceType, rs.Primary.ID)
			}
			if !client.IsNotFound(err) {
				return fmt.Errorf("checking %s %s destroyed: %w", resourceType, rs.Primary.ID, err)
			}
		}
		return nil
	}
}

// testAccImportStateIDFunc joins the named state attributes with "/" for server-generated import IDs unknown at config-authoring time.
func testAccImportStateIDFunc(resourceAddress string, attrs ...string) resource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceAddress]
		if !ok {
			return "", fmt.Errorf("resource not found in state: %s", resourceAddress)
		}
		parts := make([]string, len(attrs))
		for i, attr := range attrs {
			v, ok := rs.Primary.Attributes[attr]
			if !ok {
				return "", fmt.Errorf("attribute %q not found on %s", attr, resourceAddress)
			}
			parts[i] = v
		}
		return strings.Join(parts, "/"), nil
	}
}

// testAccWriteDeploymentPackageZip writes a minimal deployment.py package zip to a temp dir and returns its path.
func testAccWriteDeploymentPackageZip(t *testing.T, marker string) string {
	t.Helper()

	dir := t.TempDir()
	zipPath := filepath.Join(dir, "deployment_package.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("failed to create zip file: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("deployment.py")
	if err != nil {
		t.Fatalf("failed to add deployment.py to zip: %v", err)
	}
	_, err = fmt.Fprintf(w, `class Deployment:
    def __init__(self, base_directory, context):
        pass

    def request(self, data):
        # marker: %s
        return data
`, marker)
	if err != nil {
		t.Fatalf("failed to write deployment.py: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}

	return zipPath
}

// testAccWriteRequirementsFile writes a requirements.txt to a temp dir and returns its path.
func testAccWriteRequirementsFile(t *testing.T, marker string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "requirements.txt")
	content := fmt.Sprintf("# marker: %s - no additional dependencies\n", marker)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write requirements.txt: %v", err)
	}
	return path
}

func TestConfigureClient_Nil(t *testing.T) {
	var target *client.UbiOpsClient
	var diags diag.Diagnostics

	configureClient(nil, &target, &diags)

	if diags.HasError() {
		t.Fatalf("expected no errors, got: %v", diags.Errors())
	}
	if target != nil {
		t.Fatal("expected target to remain nil")
	}
}

func TestConfigureClient_ValidClient(t *testing.T) {
	c := &client.UbiOpsClient{}
	var target *client.UbiOpsClient
	var diags diag.Diagnostics

	configureClient(c, &target, &diags)

	if diags.HasError() {
		t.Fatalf("expected no errors, got: %v", diags.Errors())
	}
	if target != c {
		t.Fatal("expected target to be set to the client")
	}
}

func TestConfigureClient_WrongType(t *testing.T) {
	var target *client.UbiOpsClient
	var diags diag.Diagnostics

	configureClient("not-a-client", &target, &diags)

	if !diags.HasError() {
		t.Fatal("expected an error for wrong type")
	}
	if target != nil {
		t.Fatal("expected target to remain nil")
	}
}

func TestSetOptionalString(t *testing.T) {
	tests := []struct {
		name    string
		val     types.String
		wantSet bool
		wantVal string
	}{
		{"set value", types.StringValue("hello"), true, "hello"},
		{"null value", types.StringNull(), false, ""},
		{"unknown value", types.StringUnknown(), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{}
			setOptionalString(body, "key", tt.val)

			v, ok := body["key"]
			if ok != tt.wantSet {
				t.Fatalf("key present = %v, want %v", ok, tt.wantSet)
			}
			if ok {
				s, sOK := v.(string)
				if !sOK {
					t.Fatalf("expected string, got %T", v)
				}
				if s != tt.wantVal {
					t.Fatalf("got %q, want %q", s, tt.wantVal)
				}
			}
		})
	}
}

func TestSetOptionalInt64(t *testing.T) {
	tests := []struct {
		name    string
		val     types.Int64
		wantSet bool
		wantVal int64
	}{
		{"set value", types.Int64Value(42), true, 42},
		{"null value", types.Int64Null(), false, 0},
		{"unknown value", types.Int64Unknown(), false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{}
			setOptionalInt64(body, "key", tt.val)

			v, ok := body["key"]
			if ok != tt.wantSet {
				t.Fatalf("key present = %v, want %v", ok, tt.wantSet)
			}
			if ok {
				n, nOK := v.(int64)
				if !nOK {
					t.Fatalf("expected int64, got %T", v)
				}
				if n != tt.wantVal {
					t.Fatalf("got %d, want %d", n, tt.wantVal)
				}
			}
		})
	}
}

func TestSetOptionalBool(t *testing.T) {
	tests := []struct {
		name    string
		val     types.Bool
		wantSet bool
		wantVal bool
	}{
		{"set true", types.BoolValue(true), true, true},
		{"set false", types.BoolValue(false), true, false},
		{"null value", types.BoolNull(), false, false},
		{"unknown value", types.BoolUnknown(), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{}
			setOptionalBool(body, "key", tt.val)

			v, ok := body["key"]
			if ok != tt.wantSet {
				t.Fatalf("key present = %v, want %v", ok, tt.wantSet)
			}
			if ok {
				b, bOK := v.(bool)
				if !bOK {
					t.Fatalf("expected bool, got %T", v)
				}
				if b != tt.wantVal {
					t.Fatalf("got %v, want %v", b, tt.wantVal)
				}
			}
		})
	}
}

func TestSetChangedString(t *testing.T) {
	tests := []struct {
		name    string
		plan    types.String
		state   types.String
		wantSet bool
		wantVal string
	}{
		{"changed", types.StringValue("new"), types.StringValue("old"), true, "new"},
		{"unchanged", types.StringValue("same"), types.StringValue("same"), false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{}
			setChangedString(body, "key", tt.plan, tt.state)

			v, ok := body["key"]
			if ok != tt.wantSet {
				t.Fatalf("key present = %v, want %v", ok, tt.wantSet)
			}
			if ok {
				s, sOK := v.(string)
				if !sOK {
					t.Fatalf("expected string, got %T", v)
				}
				if s != tt.wantVal {
					t.Fatalf("got %q, want %q", s, tt.wantVal)
				}
			}
		})
	}
}

func TestSetChangedInt64(t *testing.T) {
	tests := []struct {
		name    string
		plan    types.Int64
		state   types.Int64
		wantSet bool
		wantVal int64
	}{
		{"changed", types.Int64Value(10), types.Int64Value(5), true, 10},
		{"unchanged", types.Int64Value(5), types.Int64Value(5), false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{}
			setChangedInt64(body, "key", tt.plan, tt.state)

			v, ok := body["key"]
			if ok != tt.wantSet {
				t.Fatalf("key present = %v, want %v", ok, tt.wantSet)
			}
			if ok {
				n, nOK := v.(int64)
				if !nOK {
					t.Fatalf("expected int64, got %T", v)
				}
				if n != tt.wantVal {
					t.Fatalf("got %d, want %d", n, tt.wantVal)
				}
			}
		})
	}
}

func TestSetChangedBool(t *testing.T) {
	tests := []struct {
		name    string
		plan    types.Bool
		state   types.Bool
		wantSet bool
		wantVal bool
	}{
		{"changed", types.BoolValue(true), types.BoolValue(false), true, true},
		{"unchanged", types.BoolValue(true), types.BoolValue(true), false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{}
			setChangedBool(body, "key", tt.plan, tt.state)

			v, ok := body["key"]
			if ok != tt.wantSet {
				t.Fatalf("key present = %v, want %v", ok, tt.wantSet)
			}
			if ok {
				b, bOK := v.(bool)
				if !bOK {
					t.Fatalf("expected bool, got %T", v)
				}
				if b != tt.wantVal {
					t.Fatalf("got %v, want %v", b, tt.wantVal)
				}
			}
		})
	}
}

func TestReadInt64Field(t *testing.T) {
	tests := []struct {
		name    string
		result  map[string]any
		wantVal types.Int64
	}{
		{"present", map[string]any{"key": float64(42)}, types.Int64Value(42)},
		{"missing", map[string]any{}, types.Int64Null()},
		{"nil", map[string]any{"key": nil}, types.Int64Null()},
		{"wrong type", map[string]any{"key": "not-a-number"}, types.Int64Null()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := types.Int64Null()
			readInt64Field(tt.result, "key", &target)

			if !target.Equal(tt.wantVal) {
				t.Fatalf("got %v, want %v", target, tt.wantVal)
			}
		})
	}
}

func TestReadBoolField(t *testing.T) {
	tests := []struct {
		name    string
		result  map[string]any
		wantVal types.Bool
	}{
		{"true", map[string]any{"key": true}, types.BoolValue(true)},
		{"false", map[string]any{"key": false}, types.BoolValue(false)},
		{"missing", map[string]any{}, types.BoolNull()},
		{"nil", map[string]any{"key": nil}, types.BoolNull()},
		{"wrong type", map[string]any{"key": "not-a-bool"}, types.BoolNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := types.BoolNull()
			readBoolField(tt.result, "key", &target)

			if !target.Equal(tt.wantVal) {
				t.Fatalf("got %v, want %v", target, tt.wantVal)
			}
		})
	}
}
