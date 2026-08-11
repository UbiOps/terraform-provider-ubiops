data "ubiops_instance_type_group" "example" {
  project_name = "my-project"
  id           = "group-uuid"
}

output "instance_type_group_name" {
  value = data.ubiops_instance_type_group.example.name
}
