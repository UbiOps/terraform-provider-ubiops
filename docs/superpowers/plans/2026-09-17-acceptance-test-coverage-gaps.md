# Acceptance Test Coverage Gaps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close the two real coverage gaps identified by a `go tool cover -func` breakdown of `internal/provider` (58.2% with org-level tests excluded): (1) six resources missing an `ImportState` test step, and (2) the deployment package / environment build-and-upload path (`uploadRevision`, `waitForBuild`, `computeFileSHA256`, `postVersion`, `patchDeployment`, `waitForDefaultVersion`, `isDefaultVersionNotReadyYet`) which is currently exercised by zero acceptance tests.

**Architecture:** All changes are additions to existing `*_test.go` files in `internal/provider/` plus two new fixture helpers in `helpers_test.go`. No production code changes except where a test uncovers a real bug (none currently known - if Task 8/9/10/11 surface one, fix it in the same task, matching the file's existing conventions). Fixtures (deployment package zip, requirements.txt) are generated at test runtime via `t.TempDir()` - no binary files committed to git.

**Tech Stack:** Go 1.25, `terraform-plugin-testing` v1.15.0 (`helper/resource`, `knownvalue`, `statecheck`, `tfjsonpath`, `terraform`), real UbiOps API calls gated by `TF_ACC=1`.

**Spec:** None formal - this plan implements the gap analysis from the coverage investigation in this conversation (function-level `go tool cover -func` output against `terraform-test` / `customer-succes-ubinternal`).

## Global Constraints

- Every new test MUST follow the existing skip-if-env-unset pattern: `if os.Getenv("UBIOPS_PROJECT") == "" { t.Skip(...) }`.
- Every new test MUST register a `CheckDestroy` using the existing `testAccCheckDestroyed(resourceType, pathFn)` helper - never leave orphaned real objects in a shared project.
- Match existing file conventions exactly: `// Copyright (c) Dutch Analytics B.V. 2026` + `// SPDX-License-Identifier: MPL-2.0` header, `package provider`, same import grouping, same `testAcc<Resource>ResourceConfig` naming for config-builder functions.
- These are acceptance tests: they hit the real UbiOps API. Run them with `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -run <TestName> ./internal/provider/`.
- Do NOT touch `resource_organization.go`, `resource_organization_user.go`, `resource_project.go`, `resource_project_user.go` - their 0% coverage is a permission/env-var scope issue (already diagnosed), not something this plan fixes.
- Do NOT add tests for `resource_file.go Update`, `resource_role_assignment.go Update`, `resource_service_user_token.go Update` - confirmed dead-by-design (`// All attributes require replace, so Update is never called.`), Terraform's framework never invokes them. Adding tests would require breaking the schema's `RequiresReplace` invariant artificially - not worth it.
- gofmt/golangci-lint/full test suite runs happen once at the end (Task 12), not per-task.
- Comments in new test/helper code: almost none. No step-label comments (`// Create.`, `// Update value.`, `// Delete is automatic.`); struct field names already say what a step does. At most a single-line doc comment on a new helper function, and at most one single-line comment for something genuinely non-obvious (e.g. Task B3's literal-name-vs-reference trick). This overrides any narrated comments shown in this plan's own code samples below - those predate the rule; the task briefs carry the corrected version.

---

## Phase A: Missing ImportState Coverage

**Why these six specifically:** every other resource in the package that already has an `ImportState` test step is keyed by a **user-chosen name** (`bucketName`, `deploymentName`, `roleName`, ...), so its import ID is known at config-authoring time and existing tests use a literal `ImportStateId: fmt.Sprintf("%s/%s", ...)`. These six resources are keyed by a **server-generated UUID** (`id`, distinct from any user-chosen name), which isn't known until after `apply`. That's exactly why they were skipped when the import-step pattern was copied around: it requires `ImportStateIdFunc` (pulls the real `id` out of the post-apply state), not a literal string. This phase introduces that pattern once, then applies it to all six.

### Task A0: Add the `testAccImportStateIDFunc` helper

**Files:**
- Modify: `internal/provider/helpers_test.go`

**Interfaces:**
- Produces: `func testAccImportStateIDFunc(resourceAddress string, attrs ...string) resource.ImportStateIdFunc` - used by Tasks A1-A6.

- [ ] **Step 1: Add the helper function**

Add after `testAccCheckDestroyed` (currently ends at line 88):

```go
// testAccImportStateIDFunc returns an ImportStateIdFunc that joins the named
// state attributes with "/" - for resources keyed by a server-generated id
// that isn't known until after apply (ImportStateId can't be used since it
// requires a literal string at config-authoring time).
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
```

No new imports needed - `strings` and `github.com/hashicorp/terraform-plugin-testing/terraform` are already imported in this file.

- [ ] **Step 2: Compile-check**

Run: `go build ./...`
Expected: no errors (the function is unused until Task A1 adds a caller - Go won't error on an unused package-level func, only unused local vars/imports, so this step should already pass).

- [ ] **Step 3: Commit**

```bash
git add internal/provider/helpers_test.go
git commit -m "test: add testAccImportStateIDFunc helper for UUID-keyed resources"
```

---

### Task A1: Add ImportState step to `TestAccDeploymentEnvironmentVariableResource`

**Files:**
- Modify: `internal/provider/resource_deployment_environment_variable_test.go`

**Interfaces:**
- Consumes: `testAccImportStateIDFunc` from Task A0.

- [ ] **Step 1: Add the ImportState step**

Insert between the `// Create.` step (ends line 48) and `// Update value.` step (line 50):

```go
			// ImportState.
			{
				ResourceName:      "ubiops_deployment_environment_variable.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_deployment_environment_variable.test", "project_name", "deployment_name", "id"),
				ImportStateVerify: true,
			},
```

- [ ] **Step 2: Run the test**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -run TestAccDeploymentEnvironmentVariableResource ./internal/provider/`
Expected: PASS. If it fails with a `value` mismatch during `ImportStateVerify` (the API may not echo back non-secret variable values on GET the same way it does on POST), add `ImportStateVerifyIgnore: []string{"value"}` to the step and re-run.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_deployment_environment_variable_test.go
git commit -m "test: add ImportState coverage for ubiops_deployment_environment_variable"
```

---

### Task A2: Add ImportState step to `TestAccDeploymentVersionEnvironmentVariableResource`

**Files:**
- Modify: `internal/provider/resource_deployment_version_environment_variable_test.go`

**Interfaces:**
- Consumes: `testAccImportStateIDFunc` from Task A0.

- [ ] **Step 1: Add the ImportState step**

Insert between the `// Create.` step (ends line 48) and `// Update value.` step (line 50):

```go
			// ImportState.
			{
				ResourceName:      "ubiops_deployment_version_environment_variable.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_deployment_version_environment_variable.test", "project_name", "deployment_name", "version", "id"),
				ImportStateVerify: true,
			},
```

- [ ] **Step 2: Run the test**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -run TestAccDeploymentVersionEnvironmentVariableResource ./internal/provider/`
Expected: PASS. Same `value` mismatch contingency as Task A1 applies here.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_deployment_version_environment_variable_test.go
git commit -m "test: add ImportState coverage for ubiops_deployment_version_environment_variable"
```

---

### Task A3: Add ImportState step to `TestAccInstanceTypeGroupResource`

**Files:**
- Modify: `internal/provider/resource_instance_type_group_test.go`

**Interfaces:**
- Consumes: `testAccImportStateIDFunc` from Task A0.

- [ ] **Step 1: Add the ImportState step**

Insert between the `// Create.` step (ends line 43) and `// Update name.` step (line 45):

```go
			// ImportState.
			{
				ResourceName:      "ubiops_instance_type_group.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_instance_type_group.test", "project_name", "id"),
				ImportStateVerify: true,
			},
```

- [ ] **Step 2: Run the test**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -run TestAccInstanceTypeGroupResource ./internal/provider/`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_instance_type_group_test.go
git commit -m "test: add ImportState coverage for ubiops_instance_type_group"
```

---

### Task A4: Add ImportState step to `TestAccProjectEnvironmentVariableResource`

**Files:**
- Modify: `internal/provider/resource_project_environment_variable_test.go`

**Interfaces:**
- Consumes: `testAccImportStateIDFunc` from Task A0.

- [ ] **Step 1: Add the ImportState step**

Insert between the `// Create.` step (ends line 47) and `// Update value.` step (line 49):

```go
			// ImportState.
			{
				ResourceName:      "ubiops_project_environment_variable.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_project_environment_variable.test", "project_name", "id"),
				ImportStateVerify: true,
			},
```

- [ ] **Step 2: Run the test**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -run TestAccProjectEnvironmentVariableResource ./internal/provider/`
Expected: PASS. Same `value` mismatch contingency as Task A1 applies here.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_project_environment_variable_test.go
git commit -m "test: add ImportState coverage for ubiops_project_environment_variable"
```

---

### Task A5: Add ImportState step to `TestAccServiceUserResource`

**Files:**
- Modify: `internal/provider/resource_service_user_test.go`

**Interfaces:**
- Consumes: `testAccImportStateIDFunc` from Task A0.

- [ ] **Step 1: Add the ImportState step**

Insert between the `// Create.` step (ends line 42) and `// Update.` step (line 44):

```go
			// ImportState.
			{
				ResourceName:      "ubiops_service_user.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_service_user.test", "project_name", "id"),
				ImportStateVerify: true,
			},
```

- [ ] **Step 2: Run the test**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -run TestAccServiceUserResource ./internal/provider/`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_service_user_test.go
git commit -m "test: add ImportState coverage for ubiops_service_user"
```

---

### Task A6: Add ImportState step to `TestAccRoleAssignmentResource`

**Files:**
- Modify: `internal/provider/resource_role_assignment_test.go`

**Interfaces:**
- Consumes: `testAccImportStateIDFunc` from Task A0.

- [ ] **Step 1: Add the ImportState step**

Insert between the `// Create.` step (ends line 46) and `// Delete is automatic.` comment (line 48):

```go
			// ImportState.
			{
				ResourceName:      "ubiops_role_assignment.test",
				ImportState:       true,
				ImportStateIdFunc: testAccImportStateIDFunc("ubiops_role_assignment.test", "project_name", "id"),
				ImportStateVerify: true,
			},
```

- [ ] **Step 2: Run the test**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -run TestAccRoleAssignmentResource ./internal/provider/`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_role_assignment_test.go
git commit -m "test: add ImportState coverage for ubiops_role_assignment"
```

---

## Phase B: Build & Upload Path Coverage

This is the important gap: `uploadRevision`, `waitForBuild`, `computeFileSHA256` (both `resource_deployment_version.go` and `resource_environment.go`), `postVersion`, `patchDeployment`, `waitForDefaultVersion`, and `isDefaultVersionNotReadyYet` are the actual "deploy my code" path and currently run in zero tests. These tests hit real UbiOps builds, which take real wall-clock time (expect several minutes per test, not seconds) - that's expected and correct, not a bug in the plan.

### Task B0: Add fixture-generating test helpers

**Files:**
- Modify: `internal/provider/helpers_test.go`

**Interfaces:**
- Produces: `testAccWriteDeploymentPackageZip(t *testing.T, marker string) string`, `testAccWriteRequirementsFile(t *testing.T, content string) string` - used by Tasks B1 and B2.

- [ ] **Step 1: Add "archive/zip" and "path/filepath" imports**

Modify the import block (currently lines 6-21) to add two stdlib imports:

```go
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
```

- [ ] **Step 2: Add the two fixture helpers**

Add after `testAccImportStateIDFunc` (from Task A0):

```go
// testAccWriteDeploymentPackageZip creates a minimal valid UbiOps deployment
// package (a deployment.py implementing the required Deployment class) as a
// zip file in a temp dir, and returns its path. marker is embedded as a
// comment so two calls with different markers produce different SHA-256
// hashes, which is what triggers a rebuild on update.
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

// testAccWriteRequirementsFile creates a requirements.txt in a temp dir for
// use directly as ubiops_environment's source_file (no zip needed for a
// plain requirements file per the resource's docs).
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
```

- [ ] **Step 3: Compile-check**

Run: `go build ./... && go vet ./...`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/provider/helpers_test.go
git commit -m "test: add deployment package and requirements.txt fixture helpers"
```

---

### Task B1: Add `TestAccDeploymentVersionResource_BuildFromSource`

**Files:**
- Modify: `internal/provider/resource_deployment_version_test.go`

**Interfaces:**
- Consumes: `testAccWriteDeploymentPackageZip` from Task B0.
- Covers: `uploadRevision`, `waitForBuild`, `computeFileSHA256`, `postVersion` (via `Create`), and the re-upload branch of `Update` in `resource_deployment_version.go`.

- [ ] **Step 1: Write the failing test**

Add at the end of the file:

```go
func TestAccDeploymentVersionResource_BuildFromSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	deploymentName := testAccResourceName(t)
	versionName := "v1"
	zipV1 := testAccWriteDeploymentPackageZip(t, "v1")
	zipV2 := testAccWriteDeploymentPackageZip(t, "v2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_deployment_version", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/deployments/%s/versions/%s", a["project_name"], a["deployment_name"], a["version"])
		}),
		Steps: []resource.TestStep{
			// Create: uploads deployment.py and waits for the build to succeed
			// (exercises uploadRevision, waitForBuild, computeFileSHA256).
			{
				Config: testAccDeploymentVersionResourceConfigSource(projectName, deploymentName, versionName, zipV1, 600),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("version"),
						knownvalue.StringExact(versionName),
					),
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("source_file_sha256"),
						knownvalue.NotNull(),
					),
				},
			},
			// Re-apply the identical config: source_file_sha256 must stay
			// stable (UseStateForUnknown) and produce an empty plan.
			{
				Config:   testAccDeploymentVersionResourceConfigSource(projectName, deploymentName, versionName, zipV1, 600),
				PlanOnly: true,
			},
			// Changed package content produces a new hash, re-triggering
			// uploadRevision + waitForBuild via Update().
			{
				Config: testAccDeploymentVersionResourceConfigSource(projectName, deploymentName, versionName, zipV2, 600),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment_version.test",
						tfjsonpath.New("source_file_sha256"),
						knownvalue.NotNull(),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccDeploymentVersionResourceConfigSource(projectName, deploymentName, version, zipPath string, buildTimeout int) string {
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
  project_name    = %[1]q
  deployment_name = ubiops_deployment.test.name
  version         = %[3]q
  environment     = "python3-13"
  source_file     = %[4]q
  build_timeout   = %[5]d
}
`, projectName, deploymentName, version, zipPath, buildTimeout)
}
```

- [ ] **Step 2: Run it**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -timeout 30m -run TestAccDeploymentVersionResource_BuildFromSource ./internal/provider/`
Expected: PASS, taking a few minutes (real build). If it times out, increase the `600` (seconds) argument in the two `testAccDeploymentVersionResourceConfigSource` calls - `python3-13` base-environment builds with no extra dependencies are normally well under 600s, but do not guess further without seeing the actual failure.

- [ ] **Step 3: Check coverage moved**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -timeout 30m -coverprofile=/tmp/cov_b1.out -run TestAccDeploymentVersionResource_BuildFromSource ./internal/provider/ && go tool cover -func=/tmp/cov_b1.out | grep -E "uploadRevision|waitForBuild|computeFileSHA256|postVersion"`
Expected: all four now show a percentage above 0% (each test run only covers its own test, so absolute values here will be lower than a full run - the point is they're no longer exactly 0.0%).

- [ ] **Step 4: Commit**

```bash
git add internal/provider/resource_deployment_version_test.go
git commit -m "test: exercise deployment package upload and build-wait path"
```

---

### Task B2: Add `TestAccEnvironmentResource_BuildFromSource`

**Files:**
- Modify: `internal/provider/resource_environment_test.go`

**Interfaces:**
- Consumes: `testAccWriteRequirementsFile` from Task B0.
- Covers: `uploadRevision`, `waitForBuild` (via `Create` and the re-upload branch of `Update`) in `resource_environment.go`.

- [ ] **Step 1: Write the failing test**

Add at the end of the file:

```go
func TestAccEnvironmentResource_BuildFromSource(t *testing.T) {
	projectName := os.Getenv("UBIOPS_PROJECT")
	if projectName == "" {
		t.Skip("UBIOPS_PROJECT must be set for acceptance tests")
	}

	envName := testAccResourceName(t)
	reqV1 := testAccWriteRequirementsFile(t, "v1")
	reqV2 := testAccWriteRequirementsFile(t, "v2")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		CheckDestroy: testAccCheckDestroyed("ubiops_environment", func(a map[string]string) string {
			return fmt.Sprintf("/projects/%s/environments/%s", a["project_name"], a["name"])
		}),
		Steps: []resource.TestStep{
			// Create: uploads requirements.txt and waits for the build to
			// succeed (exercises uploadRevision, waitForBuild).
			{
				Config: testAccEnvironmentResourceConfigSource(projectName, envName, reqV1, 600),
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
			// Re-apply the identical config: must produce an empty plan.
			{
				Config:   testAccEnvironmentResourceConfigSource(projectName, envName, reqV1, 600),
				PlanOnly: true,
			},
			// Changed content produces a new hash, re-triggering
			// uploadRevision + waitForBuild via Update().
			{
				Config: testAccEnvironmentResourceConfigSource(projectName, envName, reqV2, 600),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_environment.test",
						tfjsonpath.New("source_file_sha256"),
						knownvalue.NotNull(),
					),
				},
			},
			// Delete is automatic.
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
```

- [ ] **Step 2: Run it**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -timeout 30m -run TestAccEnvironmentResource_BuildFromSource ./internal/provider/`
Expected: PASS, taking a few minutes (real build).

- [ ] **Step 3: Check coverage moved**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -timeout 30m -coverprofile=/tmp/cov_b2.out -run TestAccEnvironmentResource_BuildFromSource ./internal/provider/ && go tool cover -func=/tmp/cov_b2.out | grep -A1 "resource_environment.go.*uploadRevision\|resource_environment.go.*waitForBuild"`
Expected: both functions above 0.0%.

- [ ] **Step 4: Commit**

```bash
git add internal/provider/resource_environment_test.go
git commit -m "test: exercise environment revision upload and build-wait path"
```

---

### Task B3: Add `TestAccDeploymentResource_DefaultVersionOnCreate`

**Files:**
- Modify: `internal/provider/resource_deployment_test.go`

**Interfaces:**
- Covers: `waitForDefaultVersion` in `resource_deployment.go`, called from `Create` when `default_version` is set in the initial config.

**Design note:** the deployment and its version reference each other by a **literal shared name string**, not `ubiops_deployment.test.name`. This is deliberate and matches the existing code comment on `postVersion` (`resource_deployment_version.go:243`): a `ubiops_deployment_version.test.deployment_name = ubiops_deployment.test.name` reference would create a Terraform dependency edge forcing the deployment to fully finish `Create()` (which blocks inside `waitForDefaultVersion` waiting for the version to exist) before the version resource is even allowed to start - a deadlock. Using the same literal string in both resources removes the edge so Terraform can create them without a forced order, and `waitForDefaultVersion`'s polling / `postVersion`'s create-retry cover the race either way.

- [ ] **Step 1: Write the failing test**

Add at the end of the file:

```go
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
			// Delete is automatic.
		},
	})
}

// deployment_name uses the literal deploymentName string, not a reference to
// ubiops_deployment.test.name - see this test's design note for why.
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
  deployment_name   = %[2]q
  version           = "v1"
  minimum_instances = 0
  maximum_instances = 1
}
`, projectName, deploymentName)
}
```

- [ ] **Step 2: Run it**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -timeout 10m -run TestAccDeploymentResource_DefaultVersionOnCreate ./internal/provider/`
Expected: PASS. `waitForDefaultVersion`'s default timeout is 5 minutes (`defaultVersionCreateWaitTimeout`), polling every 2 seconds - a version with no build step (`source_file` unset) should promote in seconds, well within timeout.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_deployment_test.go
git commit -m "test: exercise waitForDefaultVersion via default_version set on create"
```

---

### Task B4: Add `TestAccDeploymentResource_PromoteDefaultVersion`

**Files:**
- Modify: `internal/provider/resource_deployment_test.go`

**Interfaces:**
- Covers: `patchDeployment` and `isDefaultVersionNotReadyYet` in `resource_deployment.go`, called from `Update` when `default_version` changes on an existing deployment.

**Design note:** this exercises the code path (the PATCH-with-retry branch of `patchDeployment` always runs when `default_version` is in the update body), but whether the retry loop specifically fires depends on real timing of the live API - promoting `v2` to default in the same apply step that creates `v2` maximizes the chance of hitting the "not ready yet" retry, but this can't be forced deterministically against a real API. That's fine: even without the retry firing, `patchDeployment`'s primary PATCH call and success path are covered, which is the actual gap (currently 13.3%, driven entirely by non-default_version updates).

- [ ] **Step 1: Write the failing test**

Add at the end of the file:

```go
func TestAccDeploymentResource_PromoteDefaultVersion(t *testing.T) {
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
			// Create the deployment and its first version - the API
			// auto-promotes v1 to default with no provider involvement.
			{
				Config: testAccDeploymentResourceConfigPromote(projectName, deploymentName, false),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("default_version"),
						knownvalue.StringExact("v1"),
					),
				},
			},
			// Add v2 and promote it via an explicit default_version change on
			// the same apply - exercises patchDeployment's PATCH-with-retry.
			{
				Config: testAccDeploymentResourceConfigPromote(projectName, deploymentName, true),
				ConfigStateChecks: []statecheck.StateCheck{
					statecheck.ExpectKnownValue(
						"ubiops_deployment.test",
						tfjsonpath.New("default_version"),
						knownvalue.StringExact("v2"),
					),
				},
			},
			// Delete is automatic.
		},
	})
}

func testAccDeploymentResourceConfigPromote(projectName, deploymentName string, addV2 bool) string {
	extra := ""
	if addV2 {
		extra = fmt.Sprintf(`
resource "ubiops_deployment_version" "v2" {
  project_name      = %[1]q
  deployment_name   = ubiops_deployment.test.name
  version           = "v2"
  minimum_instances = 0
  maximum_instances = 1

  depends_on = [ubiops_deployment_version.v1]
}
`, projectName)
	}

	defaultVersion := "v1"
	if addV2 {
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
  deployment_name   = ubiops_deployment.test.name
  version           = "v1"
  minimum_instances = 0
  maximum_instances = 1
}
%[4]s
`, projectName, deploymentName, defaultVersion, extra)
}
```

Note: unlike Task B3, this config uses `ubiops_deployment.test.name` references throughout - that's fine here because `default_version` is only set to `"v1"` (already-created, always-ready) on the first apply, so `Create()`'s `waitForDefaultVersion` call resolves immediately without needing the version to be created out-of-band.

- [ ] **Step 2: Run it**

Run: `TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" go test -v -timeout 40m -run TestAccDeploymentResource_PromoteDefaultVersion ./internal/provider/`
Expected: PASS. `patchDeployment`'s retry timeout is 30 minutes (`defaultVersionUnavailableRetryTimeout`) - real promotion should resolve in well under a minute.

- [ ] **Step 3: Commit**

```bash
git add internal/provider/resource_deployment_test.go
git commit -m "test: exercise patchDeployment default_version promotion via Update"
```

---

## Phase C: Verification

### Task C0: Full coverage re-run and report

**Files:** none (verification only)

- [ ] **Step 1: Run the full acceptance suite with coverage, excluding the permission-blocked test**

Run:
```sh
TF_ACC=1 UBIOPS_API_TOKEN="Token <token>" UBIOPS_PROJECT="terraform-test" UBIOPS_ORGANIZATION="customer-succes-ubinternal" \
  go test -v -timeout 60m -coverprofile=/tmp/coverage_final.out -skip TestAccProjectResource ./internal/provider/
```
Expected: `ok`, no FAIL lines.

- [ ] **Step 2: Compare before/after coverage**

Run: `go tool cover -func=/tmp/coverage_final.out | tail -1`
Expected: total statement coverage higher than the 58.2% baseline recorded before this plan (exact number depends on real API timing/build variance - report whatever it is, don't target a specific number).

- [ ] **Step 3: Confirm the specific target functions moved**

Run: `go tool cover -func=/tmp/coverage_final.out | grep -E "uploadRevision|waitForBuild|computeFileSHA256|postVersion|patchDeployment|waitForDefaultVersion|isDefaultVersionNotReadyYet|ImportState"`
Expected: none of these show `0.0%` anymore (except any `ImportState` entries in the four intentionally-excluded org/project files, which remain out of scope per Global Constraints).

- [ ] **Step 4: Run gofmt/lint once**

Run: `gofmt -l internal/provider/` (expect empty output) then `golangci-lint run` (expect no new findings).

- [ ] **Step 5: Final commit if formatting touched anything**

```bash
git add -A
git commit -m "test: gofmt cleanup after coverage gap closure"
```
(skip this step entirely if Step 4 made no changes)
