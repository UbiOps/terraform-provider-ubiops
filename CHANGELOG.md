<!-- markdownlint-disable MD024 MD041 -->

## 0.1.0 (Unreleased)

FEATURES:

- **New Resource:** `ubiops_project` — manage UbiOps projects
- **New Resource:** `ubiops_deployment` — manage deployments
- **New Resource:** `ubiops_deployment_version` — manage deployment versions,
  including deployment package upload via `source_file`
- **New Resource:** `ubiops_environment` — manage custom environments
- **New Resource:** `ubiops_project_environment_variable` — manage project-level
  environment variables
- **New Resource:** `ubiops_deployment_environment_variable` — manage
  deployment-level environment variables
- **New Resource:** `ubiops_deployment_version_environment_variable` — manage
  version-level environment variables
- **New Resource:** `ubiops_bucket` — manage storage buckets
- **New Resource:** `ubiops_file` — manage files in buckets
- **New Resource:** `ubiops_pipeline` — manage pipelines
- **New Resource:** `ubiops_pipeline_version` — manage pipeline versions
- **New Resource:** `ubiops_service` — manage services
- **New Resource:** `ubiops_service_user` — manage service users
- **New Resource:** `ubiops_service_user_token` — manage service user tokens
- **New Resource:** `ubiops_role` — manage custom roles
- **New Resource:** `ubiops_role_assignment` — manage role assignments
- **New Resource:** `ubiops_webhook` — manage webhooks
- **New Resource:** `ubiops_request_schedule` — manage request schedules
- **New Resource:** `ubiops_instance_type_group` — manage instance type groups
- **New Resource:** `ubiops_metric` — manage custom metrics
- **New Resource:** `ubiops_organization` — manage organizations
- **New Resource:** `ubiops_organization_user` — manage organization users
- **New Resource:** `ubiops_project_user` — manage project users
- **New Data Source:** `ubiops_project` — read project details
- **New Data Source:** `ubiops_deployment` — read deployment details
- **New Data Source:** `ubiops_environment` — read environment details
- **New Data Source:** `ubiops_pipeline` — read pipeline details
- **New Data Source:** `ubiops_bucket` — read bucket details
- **New Data Source:** `ubiops_service` — read service details
- **New Data Source:** `ubiops_instance_type_group` — read instance type group
  details

ENHANCEMENTS:

- `ubiops_deployment`: `default_version` is now settable (previously
  read-only), matching `ubiops_pipeline`. Enables promoting a new
  `ubiops_deployment_version` to receive traffic without recreating the
  deployment — the basis for zero-downtime version rollout and rollback.
