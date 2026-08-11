resource "ubiops_service_user_token" "example" {
  project_name    = "my-project"
  service_user_id = ubiops_service_user.example.id
}
