resource "ubiops_webhook" "example" {
  project_name = "my-project"
  name         = "my-webhook"
  url          = "https://example.com/webhook"
  event        = "deployment_request_completed"
  object_type  = "deployment"
  object_name  = "my-deployment"
  enabled      = true
}
