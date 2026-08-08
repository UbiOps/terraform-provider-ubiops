resource "ubiops_request_schedule" "example" {
  project_name = "my-project"
  name         = "my-schedule"
  object_type  = "deployment"
  object_name  = "my-deployment"
  version      = "v1"
  schedule     = "0 */6 * * *"
  enabled      = true

  request_data_json = jsonencode({
    input = "scheduled request"
  })
}
