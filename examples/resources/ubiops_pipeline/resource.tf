resource "ubiops_pipeline" "example" {
  project_name = "my-project"
  name         = "my-pipeline"
  input_type   = "structured"
  output_type  = "structured"

  input_fields {
    name      = "input"
    data_type = "string"
  }

  output_fields {
    name      = "output"
    data_type = "string"
  }

  labels = {
    environment = "production"
  }
}
