resource "ubiops_organization_user" "example" {
  organization_name = "my-organization"
  email             = "user@example.com"
  admin             = false
}
