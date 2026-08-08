data "ubiops_environment" "example" {
  project_name = "my-project"
  name         = "python3-12"
}

output "environment_id" {
  value = data.ubiops_environment.example.id
}
