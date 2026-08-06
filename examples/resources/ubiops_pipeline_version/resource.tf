resource "ubiops_pipeline_version" "example" {
  project_name  = "my-project"
  pipeline_name = "my-pipeline"
  version       = "v1"

  request_retention_mode = "full"

  objects = [
    {
      name           = "my-deployment"
      reference_type = "deployment"
      reference_name = "my-deployment"
      version        = "v1"
    }
  ]

  attachments = [
    {
      destination_name = "my-deployment"
      sources = [
        {
          source_name = "pipeline_start"
          mapping = [
            {
              source_field_name      = "input"
              destination_field_name = "input"
            }
          ]
        }
      ]
    },
    {
      destination_name = "pipeline_end"
      sources = [
        {
          source_name = "my-deployment"
          mapping = [
            {
              source_field_name      = "output"
              destination_field_name = "output"
            }
          ]
        }
      ]
    }
  ]
}
