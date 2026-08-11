resource "ubiops_role" "example" {
  project_name = "my-project"
  name         = "my-custom-role"
  permissions = [
    "deployments.list",
    "deployments.get",
    "deployments.create",
  ]
}
