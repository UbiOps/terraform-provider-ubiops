resource "ubiops_role_assignment" "example" {
  project_name  = "my-project"
  role          = "project-admin"
  assignee      = ubiops_service_user.example.id
  assignee_type = "user"
  resource      = "my-project"
  resource_type = "project"
}
