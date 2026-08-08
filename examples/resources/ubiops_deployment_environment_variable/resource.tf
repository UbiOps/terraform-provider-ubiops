resource "ubiops_deployment_environment_variable" "example" {
  project_name    = "my-project"
  deployment_name = "my-deployment"
  name            = "MODEL_PATH"
  value           = "/models/v1"
}
