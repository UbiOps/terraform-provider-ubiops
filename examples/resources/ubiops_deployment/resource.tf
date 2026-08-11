resource "ubiops_deployment" "example" {
  project_name = "my-project"
  name         = "my-deployment"
  description  = "An example deployment"
  input_type   = "structured"
  output_type  = "structured"

  input_fields = [
    {
      name      = "input"
      data_type = "string"
    }
  ]

  output_fields = [
    {
      name      = "output"
      data_type = "string"
    }
  ]

  labels = {
    environment = "production"
  }
}
