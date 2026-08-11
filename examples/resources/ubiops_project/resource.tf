resource "ubiops_project" "example" {
  name              = "my-project"
  organization_name = "my-organization"

  labels = {
    environment = "production"
  }
}
