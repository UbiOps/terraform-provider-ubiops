# UbiOps Terraform Provider

**THE UBIOPS TERRAFORM PROVIDER IS IN EARLY ALPHA DEVELOPMENT.** We do not
recommend using the provider in production yet. Let us know if you want to help
us test this!

The Terraform provider for [UbiOps](UbiOps), a platform for ML model serving and
orchestration.

**See the
[official documentation](https://registry.terraform.io/providers/ubiops/ubiops/latest/docs)
to learn about all the possible services and resources.**

## Quick start

- Sign up for UbiOps
- Create your API token
- Create a file called `main.tf` with the content below

  ```terraform
  terraform {
    required_version = ">= 1.0"

    required_providers {
      ubiops = {
        source  = "registry.terraform.io/ubiops/ubiops"
        version = "~> 0.1"
      }
    }
  }

  provider "ubiops" {
    # api_token = "Token abc123"                   # Or set UBIOPS_API_TOKEN
    # base_url  = "https://api.ubiops.com/v2.1"   # Or set UBIOPS_BASE_URL
  }

  resource "ubiops_project" "example" {
    name              = "my-project"
    organization_name = "my-organization"
  }

  resource "ubiops_deployment" "example" {
    project_name = ubiops_project.example.name
    name         = "my-deployment"
    description  = "A simple deployment"
    input_type   = "structured"
    output_type  = "structured"

    input_fields = [
      { name = "input", data_type = "string" }
    ]

    output_fields = [
      { name = "output", data_type = "string" }
    ]
  }

  resource "ubiops_deployment_version" "example" {
    project_name    = ubiops_project.example.name
    deployment_name = ubiops_deployment.example.name
    version         = "v1"
    environment     = "python3-13"
    source_file     = "deployment_package.zip"
  }
  ```

- Run these commands in your terminal:

  ```sh
  terraform init
  terraform plan
  terraform apply
  ```

Voilà, a UbiOps deployment.

## A word of caution

Recreating stateful resources with Terraform will possibly **delete** the
resource and all its data before creating it again. Whenever the Terraform plan
indicates that a resource will be **deleted** or **replaced**, a catastrophic
action is possibly about to happen.

Some properties, like **project** and the **resource name**, cannot be changed
and it will trigger a resource replacement.

## Contributing

Bug reports and patches are very welcome, please post them as GitHub issues and
pull requests at <https://github.com/ubiops/ubiops>. Please review the guides
below.

- [Contributing guidelines](CONTRIBUTING.md)
- [Code of conduct](.github/CODE_OF_CONDUCT.md)

Please see our [security](SECURITY.md) policy to report any possible
vulnerabilities or serious issues.

## License

terraform-provider-ubiops is licensed under the MPL license. Full license text
is available in the [LICENSE](LICENSE) file. Please note that the project
explicitly does not require a CLA (Contributor License Agreement) from its
contributors.
