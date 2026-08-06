resource "ubiops_environment" "example" {
  project_name     = "my-project"
  name             = "my-environment"
  base_environment = "python3-12"
  description      = "Custom environment with additional packages"

  labels = {
    team = "data-science"
  }

  # Upload a zip containing requirements.txt to install packages.
  # The provider waits for the build to complete before returning.
  source_file        = "./packages/requirements.zip"
  source_file_sha256 = filesha256("./packages/requirements.zip")
  build_timeout      = 1800
}
