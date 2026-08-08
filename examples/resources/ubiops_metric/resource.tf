resource "ubiops_metric" "example" {
  project_name = "my-project"
  name         = "my-custom-metric"
  description  = "Tracks custom events"
  metric_type  = "GAUGE"
  unit         = "count"
  labels       = ["environment", "region"]
}
