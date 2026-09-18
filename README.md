# UbiOps Terraform Provider

[![Tests](https://github.com/UbiOps/terraform-provider-ubiops/actions/workflows/test.yml/badge.svg)](https://github.com/UbiOps/terraform-provider-ubiops/actions/workflows/test.yml)
[![License: MPL 2.0](https://img.shields.io/badge/License-MPL%202.0-brightgreen.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/UbiOps/terraform-provider-ubiops.svg)](https://pkg.go.dev/github.com/UbiOps/terraform-provider-ubiops)

The Terraform provider for [UbiOps](https://ubiops.com), a platform for ML
model serving and orchestration.

**See the
[official documentation](https://registry.terraform.io/providers/ubiops/ubiops/latest/docs)
to learn about all the possible services and resources.**

## Quick start

1. [Sign up for UbiOps](https://app.ubiops.com/sign-up/) and create your API
   token.
2. Create a file called `main.tf`:

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
     # api_token = "Token abc123"                  # Or set UBIOPS_API_TOKEN
     # base_url  = "https://api.ubiops.com/v2.1"    # Or set UBIOPS_BASE_URL
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

   resource "ubiops_instance_type_group" "example" {
     project_name = ubiops_project.example.name
     name         = "my-instance-group"

     instance_types_json = jsonencode([
       { id = "256mb", priority = 1 }
     ])
   }

   resource "ubiops_deployment_version" "example" {
     project_name             = ubiops_project.example.name
     deployment_name          = ubiops_deployment.example.name
     version                  = "v1"
     environment              = "python3-13"
     instance_type_group_name = ubiops_instance_type_group.example.name
     source_file              = "deployment_package.zip"
   }
   ```

3. Run:

   ```sh
   terraform init
   terraform plan
   terraform apply
   ```

Voilà, a UbiOps deployment.

## A word of caution

Recreating a stateful resource with Terraform can **delete** it (and its data)
before creating the replacement. Whenever `terraform plan` shows a resource as
**deleted** or **replaced**, review the plan carefully before applying - that
step is destructive and not reversible.

Some properties, like **project** and the **resource name**, can't be changed
in place and will trigger this kind of replacement.

## Contributing

Bug reports and patches are very welcome, please post them as GitHub issues and
pull requests at <https://github.com/UbiOps/terraform-provider-ubiops>. Please
review the guides below.

- [Contributing guidelines](CONTRIBUTING.md)
- [Code of conduct](.github/CODE_OF_CONDUCT.md)

Please see our [security](SECURITY.md) policy to report any possible
vulnerabilities or serious issues.

## License

terraform-provider-ubiops is licensed under the MPL license. Full license text
is available in the [LICENSE](LICENSE) file. Please note that the project
explicitly does not require a CLA (Contributor License Agreement) from its
contributors.
