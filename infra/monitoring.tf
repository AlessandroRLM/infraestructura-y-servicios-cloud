# ---------------------------------------------------------------------------
# Notification channel
# ---------------------------------------------------------------------------

resource "google_monitoring_notification_channel" "email" {
  project      = var.project_id
  display_name = "Ops Email"
  type         = "email"

  labels = {
    email_address = var.alert_email
  }
}

# ---------------------------------------------------------------------------
# Uptime check — HTTPS GET /healthz on the public hostname (frontend/nginx)
# API-level availability is covered by the RPC error-rate alert below.
# ---------------------------------------------------------------------------

resource "google_monitoring_uptime_check_config" "app_healthz" {
  project      = var.project_id
  display_name = "App /healthz"
  period       = "60s"
  timeout      = "10s"

  http_check {
    path         = "/healthz"
    port         = 443
    use_ssl      = true
    validate_ssl = true
  }

  monitored_resource {
    type = "uptime_url"
    labels = {
      project_id = var.project_id
      host       = var.app_host
    }
  }
}


# ---------------------------------------------------------------------------
# Alert: CPU utilization > 70% sustained for 5 min
# ---------------------------------------------------------------------------

resource "google_monitoring_alert_policy" "cpu_high" {
  project      = var.project_id
  display_name = "GKE Node CPU > 70%"
  combiner     = "OR"

  conditions {
    display_name = "Node CPU utilization above threshold"

    condition_threshold {
      filter          = "resource.type = \"k8s_node\" AND metric.type = \"kubernetes.io/node/cpu/allocatable_utilization\""
      comparison      = "COMPARISON_GT"
      threshold_value = 0.7
      duration        = "300s"

      aggregations {
        alignment_period     = "60s"
        per_series_aligner   = "ALIGN_MEAN"
        cross_series_reducer = "REDUCE_MEAN"
        group_by_fields      = ["resource.label.node_name"]
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]

  alert_strategy {
    auto_close = "1800s"
  }
}

# ---------------------------------------------------------------------------
# Alert: Public app endpoint down (uptime check on /healthz → frontend/nginx)
# API-level availability is covered by the RPC error-rate alert.
# ---------------------------------------------------------------------------

resource "google_monitoring_alert_policy" "service_down" {
  project      = var.project_id
  display_name = "Public App Endpoint Down (uptime check)"
  combiner     = "OR"

  conditions {
    display_name = "Public app endpoint unreachable from 2+ prober locations"

    condition_threshold {
      filter          = "resource.type = \"uptime_url\" AND metric.type = \"monitoring.googleapis.com/uptime_check/check_passed\" AND metric.label.check_id = \"${google_monitoring_uptime_check_config.app_healthz.uptime_check_id}\""
      comparison      = "COMPARISON_GT"
      threshold_value = 1
      duration        = "60s"

      aggregations {
        alignment_period     = "1200s"
        per_series_aligner   = "ALIGN_NEXT_OLDER"
        cross_series_reducer = "REDUCE_COUNT_FALSE"
        group_by_fields      = ["resource.label.host"]
      }
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]

  alert_strategy {
    auto_close = "1800s"
  }
}

# ---------------------------------------------------------------------------
# Alert: RPC error rate from GMP metric
# Uses condition_prometheus_query_language — GA in google ~> 6.0
# ---------------------------------------------------------------------------

resource "google_monitoring_alert_policy" "rpc_error_rate" {
  project      = var.project_id
  display_name = "API RPC Error Rate"
  combiner     = "OR"

  conditions {
    display_name = "RPC error rate > 5% sustained"

    condition_prometheus_query_language {
      query               = <<-EOQ
        (
          sum(rate(academico_rpc_requests_total{code!="ok"}[5m]))
          /
          sum(rate(academico_rpc_requests_total[5m]))
        ) > 0.05
      EOQ
      duration            = "120s"
      evaluation_interval = "60s"
    }
  }

  notification_channels = [google_monitoring_notification_channel.email.id]

  alert_strategy {
    auto_close = "1800s"
  }
}

# ---------------------------------------------------------------------------
# Billing budget with threshold notifications
# ---------------------------------------------------------------------------

resource "google_billing_budget" "monthly" {
  billing_account = var.billing_account_id
  display_name    = "IYSC Monthly Budget"

  budget_filter {
    projects = ["projects/${data.google_project.project.number}"]
  }

  amount {
    specified_amount {
      currency_code = "USD"
      units         = tostring(var.monthly_budget_usd)
    }
  }

  threshold_rules {
    threshold_percent = 0.5
  }

  threshold_rules {
    threshold_percent = 0.9
  }

  threshold_rules {
    threshold_percent = 1.0
    spend_basis       = "FORECASTED_SPEND"
  }

  all_updates_rule {
    monitoring_notification_channels = [
      google_monitoring_notification_channel.email.id,
    ]
    disable_default_iam_recipients = false
  }
}

# ---------------------------------------------------------------------------
# Dashboard: Infrastructure (node CPU, memory, disk)
# ---------------------------------------------------------------------------

resource "google_monitoring_dashboard" "infra" {
  project = var.project_id
  dashboard_json = jsonencode({
    displayName = "IYSC Infrastructure"
    gridLayout = {
      columns = "2"
      widgets = [
        {
          title = "Node CPU Allocatable Utilization"
          xyChart = {
            dataSets = [{
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "resource.type=\"k8s_node\" AND metric.type=\"kubernetes.io/node/cpu/allocatable_utilization\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MEAN"
                    crossSeriesReducer = "REDUCE_MEAN"
                    groupByFields      = ["resource.label.node_name"]
                  }
                }
              }
              plotType = "LINE"
            }]
          }
        },
        {
          title = "Node Memory Allocatable Utilization"
          xyChart = {
            dataSets = [{
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "resource.type=\"k8s_node\" AND metric.type=\"kubernetes.io/node/memory/allocatable_utilization\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MEAN"
                    crossSeriesReducer = "REDUCE_MEAN"
                    groupByFields      = ["resource.label.node_name"]
                  }
                }
              }
              plotType = "LINE"
            }]
          }
        },
        {
          title = "Container CPU Limit Utilization"
          xyChart = {
            dataSets = [{
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "resource.type=\"k8s_container\" AND metric.type=\"kubernetes.io/container/cpu/limit_utilization\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MEAN"
                    crossSeriesReducer = "REDUCE_MEAN"
                    groupByFields      = ["resource.label.container_name"]
                  }
                }
              }
              plotType = "LINE"
            }]
          }
        },
        {
          title = "Container Memory Limit Utilization"
          xyChart = {
            dataSets = [{
              timeSeriesQuery = {
                timeSeriesFilter = {
                  filter = "resource.type=\"k8s_container\" AND metric.type=\"kubernetes.io/container/memory/limit_utilization\""
                  aggregation = {
                    alignmentPeriod    = "60s"
                    perSeriesAligner   = "ALIGN_MEAN"
                    crossSeriesReducer = "REDUCE_MEAN"
                    groupByFields      = ["resource.label.container_name"]
                  }
                }
              }
              plotType = "LINE"
            }]
          }
        }
      ]
    }
  })
}

# ---------------------------------------------------------------------------
# Dashboard: Application (RPC rate, error rate, p95 latency, rejections)
# GMP metrics appear as prometheus.googleapis.com/* in Cloud Monitoring
# ---------------------------------------------------------------------------

resource "google_monitoring_dashboard" "app" {
  project = var.project_id
  dashboard_json = jsonencode({
    displayName = "IYSC Application"
    gridLayout = {
      columns = "2"
      widgets = [
        {
          title = "RPC Request Rate (req/s)"
          xyChart = {
            dataSets = [{
              timeSeriesQuery = {
                prometheusQuery = "sum by (service, method)(rate(academico_rpc_requests_total[5m]))"
              }
              plotType = "LINE"
            }]
          }
        },
        {
          title = "RPC Error Rate (errors/s)"
          xyChart = {
            dataSets = [{
              timeSeriesQuery = {
                prometheusQuery = "sum by (service, code)(rate(academico_rpc_requests_total{code!=\"ok\"}[5m]))"
              }
              plotType = "LINE"
            }]
          }
        },
        {
          title = "RPC Latency p95 (seconds)"
          xyChart = {
            dataSets = [{
              timeSeriesQuery = {
                prometheusQuery = "histogram_quantile(0.95, sum by (le)(rate(academico_rpc_duration_seconds_bucket[5m])))"
              }
              plotType = "LINE"
            }]
          }
        },
        {
          title = "Enrollment Rejection Counters"
          xyChart = {
            dataSets = [
              {
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"prometheus_target\" AND metric.type=\"prometheus.googleapis.com/academico_section_full_total/counter\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_RATE"
                    }
                  }
                }
                plotType       = "LINE"
                legendTemplate = "section_full"
              },
              {
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"prometheus_target\" AND metric.type=\"prometheus.googleapis.com/academico_section_lock_timeout_total/counter\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_RATE"
                    }
                  }
                }
                plotType       = "LINE"
                legendTemplate = "lock_timeout"
              },
              {
                timeSeriesQuery = {
                  timeSeriesFilter = {
                    filter = "resource.type=\"prometheus_target\" AND metric.type=\"prometheus.googleapis.com/academico_admission_saturated_total/counter\""
                    aggregation = {
                      alignmentPeriod  = "60s"
                      perSeriesAligner = "ALIGN_RATE"
                    }
                  }
                }
                plotType       = "LINE"
                legendTemplate = "admission_saturated"
              }
            ]
          }
        }
      ]
    }
  })
}

# ---------------------------------------------------------------------------
# Dashboard: Costs (informational — billing data is not a Monitoring metric)
# ---------------------------------------------------------------------------

resource "google_monitoring_dashboard" "costs" {
  project = var.project_id
  dashboard_json = jsonencode({
    displayName = "IYSC Costs"
    gridLayout = {
      columns = "1"
      widgets = [
        {
          title = "Cost Monitoring Note"
          text = {
            content = "Real-time billing data is not available as a Cloud Monitoring metric. Use the GCP Billing console (https://console.cloud.google.com/billing) or export billing data to BigQuery for cost analytics. Budget alert thresholds (50%/90%/100%) are managed via the `google_billing_budget.monthly` Terraform resource."
            format  = "RAW"
          }
        }
      ]
    }
  })
}
