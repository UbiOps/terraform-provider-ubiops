resource "ubiops_service" "example" {
  project_name = "my-project"
  name         = "my-service"
  deployment   = "my-deployment"
  version      = "v1"
  port         = 8080

  authentication_required = true
}
