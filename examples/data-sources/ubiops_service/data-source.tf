data "ubiops_service" "example" {
  project_name = "my-project"
  name         = "my-service"
}

output "service_id" {
  value = data.ubiops_service.example.id
}
