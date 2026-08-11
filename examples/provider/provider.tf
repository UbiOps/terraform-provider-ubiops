terraform {
  required_version = ">= 1.0"

  required_providers {
    ubiops = {
      source  = "registry.terraform.io/ubiops/ubiops"
      version = "~> 0.1"
    }
  }
}

# Configure the UbiOps provider
provider "ubiops" {
  # api_token = "abc123"  # raw token — the provider prepends "Token " automatically
  # base_url  = "https://api.ubiops.com/v2.1"   # Or set UBIOPS_BASE_URL
}
