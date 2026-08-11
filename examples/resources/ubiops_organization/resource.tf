resource "ubiops_organization" "example" {
  name                             = "my-organization"
  subscription                     = "professional"
  two_factor_authentication_forced = true
}
