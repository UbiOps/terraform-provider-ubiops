data "ubiops_deployment" "example" {
  project_name = "my-project"
  name         = "my-deployment"
}

output "deployment_id" {
  value = data.ubiops_deployment.example.id
}
