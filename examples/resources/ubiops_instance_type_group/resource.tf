resource "ubiops_instance_type_group" "example" {
  project_name = "my-project"
  name         = "my-instance-group"

  instance_types_json = jsonencode([
    {
      id               = "256mb"
      priority         = 1
      schedule_timeout = 300
    },
    {
      id               = "512mb"
      priority         = 2
      schedule_timeout = 600
    }
  ])
}
