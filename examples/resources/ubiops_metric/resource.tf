resource "ubiops_metric" "example" {
  project_name = "my-project"
  name         = "custom.my-custom-metric"
  description  = "Tracks custom events"
  metric_type  = "gauge"
  unit         = "count"
  labels       = ["environment", "region"]
}
