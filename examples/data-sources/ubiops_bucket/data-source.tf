data "ubiops_bucket" "example" {
  project_name = "my-project"
  name         = "my-bucket"
}

output "bucket_id" {
  value = data.ubiops_bucket.example.id
}
