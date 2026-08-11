resource "ubiops_project_environment_variable" "example" {
  project_name = "my-project"
  name         = "API_KEY"
  value        = "my-secret-key"
  secret       = true
}
