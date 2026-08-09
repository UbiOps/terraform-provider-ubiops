resource "ubiops_deployment_version" "example" {
  project_name    = "my-project"
  deployment_name = "my-deployment"
  version         = "v1"
  environment     = "python3-13"
  source_file     = "deployment_package.zip"

  minimum_instances      = 0
  maximum_instances      = 5
  maximum_idle_time      = 300
  request_retention_mode = "full"

  labels = {
    environment = "production"
  }
}
