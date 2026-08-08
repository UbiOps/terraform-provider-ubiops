resource "ubiops_bucket" "example" {
  project_name    = "my-project"
  name            = "my-bucket"
  description     = "A storage bucket"
  bucket_provider = "ubiops"
  ttl             = 86400

  labels = {
    environment = "production"
  }
}
