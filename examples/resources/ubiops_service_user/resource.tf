resource "ubiops_service_user" "example" {
  project_name = "my-project"
  name         = "my-service-user"
  description  = "Service user for CI/CD"
}
