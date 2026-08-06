resource "ubiops_deployment_version_environment_variable" "example" {
  project_name    = "my-project"
  deployment_name = "my-deployment"
  version         = "v1"
  name            = "BATCH_SIZE"
  value           = "32"
}
