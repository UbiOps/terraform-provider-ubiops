data "ubiops_project" "example" {
  name = "my-project"
}

output "project_id" {
  value = data.ubiops_project.example.id
}
