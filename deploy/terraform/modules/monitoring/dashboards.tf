# =============================================================================
# Grafana Dashboard ConfigMaps
# Auto-loaded by Grafana via sidecar
# =============================================================================

resource "kubernetes_config_map" "grafana_dashboard_overview" {
  metadata {
    name      = "grafana-dashboard-clario360-overview"
    namespace = var.namespace
    labels = {
      grafana_dashboard = "1"
    }
  }

  data = {
    "clario360-overview.json" = jsonencode({
      annotations = { list = [] }
      editable    = true
      title       = "Clario 360 - Platform Overview"
      uid         = "clario360-overview"
      panels = [
        {
          title      = "Request Rate (all services)"
          type       = "timeseries"
          gridPos    = { h = 8, w = 12, x = 0, y = 0 }
          datasource = { type = "prometheus" }
          targets = [{
            expr         = "sum(rate(http_requests_total{namespace=\"clario360\"}[5m])) by (service)"
            legendFormat = "{{service}}"
          }]
        },
        {
          title      = "Error Rate"
          type       = "timeseries"
          gridPos    = { h = 8, w = 12, x = 12, y = 0 }
          datasource = { type = "prometheus" }
          targets = [{
            expr         = "sum(rate(http_requests_total{namespace=\"clario360\",code=~\"5..\"}[5m])) by (service)"
            legendFormat = "{{service}}"
          }]
        },
        {
          title      = "P99 Latency"
          type       = "timeseries"
          gridPos    = { h = 8, w = 12, x = 0, y = 8 }
          datasource = { type = "prometheus" }
          targets = [{
            expr         = "histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{namespace=\"clario360\"}[5m])) by (service, le))"
            legendFormat = "{{service}}"
          }]
        },
        {
          title      = "Pod Status"
          type       = "stat"
          gridPos    = { h = 8, w = 12, x = 12, y = 8 }
          datasource = { type = "prometheus" }
          targets = [{
            expr         = "count(kube_pod_status_phase{namespace=\"clario360\",phase=\"Running\"})"
            legendFormat = "Running Pods"
          }]
        },
      ]
      time = { from = "now-1h", to = "now" }
      refresh = "30s"
    })
  }

  depends_on = [helm_release.prometheus_stack]
}

resource "kubernetes_config_map" "grafana_dashboard_database" {
  metadata {
    name      = "grafana-dashboard-clario360-database"
    namespace = var.namespace
    labels = {
      grafana_dashboard = "1"
    }
  }

  data = {
    "clario360-database.json" = jsonencode({
      title = "Clario 360 - Database"
      uid   = "clario360-database"
      panels = [
        {
          title      = "Active Connections"
          type       = "timeseries"
          gridPos    = { h = 8, w = 12, x = 0, y = 0 }
          datasource = { type = "prometheus" }
          targets = [{
            expr         = "pg_stat_activity_count{datname=~\"clario360_.*\"}"
            legendFormat = "{{datname}}"
          }]
        },
        {
          title      = "Query Duration (P95)"
          type       = "timeseries"
          gridPos    = { h = 8, w = 12, x = 12, y = 0 }
          datasource = { type = "prometheus" }
          targets = [{
            expr         = "histogram_quantile(0.95, rate(pg_stat_statements_seconds_bucket[5m]))"
            legendFormat = "P95"
          }]
        },
        {
          title      = "Cache Hit Ratio"
          type       = "gauge"
          gridPos    = { h = 8, w = 12, x = 0, y = 8 }
          datasource = { type = "prometheus" }
          targets = [{
            expr = "pg_stat_database_blks_hit{datname=~\"clario360_.*\"} / (pg_stat_database_blks_hit{datname=~\"clario360_.*\"} + pg_stat_database_blks_read{datname=~\"clario360_.*\"})"
          }]
        },
      ]
      time    = { from = "now-1h", to = "now" }
      refresh = "30s"
    })
  }

  depends_on = [helm_release.prometheus_stack]
}
