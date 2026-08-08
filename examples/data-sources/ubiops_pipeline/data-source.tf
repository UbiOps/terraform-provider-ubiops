data "ubiops_pipeline" "example" {
  project_name = "my-project"
  name         = "my-pipeline"
}

output "pipeline_id" {
  value = data.ubiops_pipeline.example.id
}
