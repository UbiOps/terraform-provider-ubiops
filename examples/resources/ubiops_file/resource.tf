resource "ubiops_file" "example" {
  project_name = "my-project"
  bucket_name  = "my-bucket"
  file         = "data/input.json"
}
