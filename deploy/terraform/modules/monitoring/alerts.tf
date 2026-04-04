# =============================================================================
# PrometheusRule Alerting Configuration
# Critical alerts for platform health, performance, and security
# =============================================================================

resource "kubernetes_manifest" "prometheus_alerts" {
  manifest = {
    apiVersion = "monitoring.coreos.com/v1"
    kind       = "PrometheusRule"
    metadata = {
      name      = "clario360-alerts"
      namespace = var.namespace
      labels = {
        "app.kubernetes.io/part-of" = "clario360"
        release                     = "kube-prometheus-stack"
      }
    }
    spec = {
      groups = [
        {
          name = "clario360.service.alerts"
          rules = [
            {
              alert = "HighErrorRate"
              expr  = "sum(rate(http_requests_total{namespace=\"clario360\",code=~\"5..\"}[5m])) by (service) / sum(rate(http_requests_total{namespace=\"clario360\"}[5m])) by (service) > 0.05"
              for   = "5m"
              labels = {
                severity = "critical"
              }
              annotations = {
                summary     = "High error rate on {{ $labels.service }}"
                description = "Service {{ $labels.service }} has >5% error rate for 5 minutes. Current: {{ $value | humanizePercentage }}"
              }
            },
            {
              alert = "HighLatency"
              expr  = "histogram_quantile(0.99, sum(rate(http_request_duration_seconds_bucket{namespace=\"clario360\"}[5m])) by (service, le)) > 2"
              for   = "5m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "High P99 latency on {{ $labels.service }}"
                description = "Service {{ $labels.service }} P99 latency >2s for 5 minutes. Current: {{ $value | humanizeDuration }}"
              }
            },
            {
              alert = "PodCrashLooping"
              expr  = "rate(kube_pod_container_status_restarts_total{namespace=\"clario360\"}[15m]) > 0"
              for   = "15m"
              labels = {
                severity = "critical"
              }
              annotations = {
                summary     = "Pod crash-looping: {{ $labels.pod }}"
                description = "Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} is restarting frequently."
              }
            },
            {
              alert = "PodNotReady"
              expr  = "kube_pod_status_ready{namespace=\"clario360\",condition=\"true\"} == 0"
              for   = "10m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "Pod not ready: {{ $labels.pod }}"
                description = "Pod {{ $labels.pod }} has been in not-ready state for 10 minutes."
              }
            },
          ]
        },
        {
          name = "clario360.database.alerts"
          rules = [
            {
              alert = "DatabaseConnectionsHigh"
              expr  = "pg_stat_activity_count{datname=~\"clario360_.*\"} > 80"
              for   = "5m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "High database connections on {{ $labels.datname }}"
                description = "Database {{ $labels.datname }} has {{ $value }} active connections (threshold: 80)."
              }
            },
            {
              alert = "DatabaseSlowQueries"
              expr  = "rate(pg_stat_statements_seconds_sum[5m]) / rate(pg_stat_statements_seconds_count[5m]) > 1"
              for   = "10m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "Slow database queries detected"
                description = "Average query duration exceeds 1 second for 10 minutes."
              }
            },
          ]
        },
        {
          name = "clario360.kafka.alerts"
          rules = [
            {
              alert = "KafkaConsumerLag"
              expr  = "kafka_consumergroup_lag_sum > 10000"
              for   = "10m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "High Kafka consumer lag: {{ $labels.consumergroup }}"
                description = "Consumer group {{ $labels.consumergroup }} has lag of {{ $value }} messages."
              }
            },
            {
              alert = "KafkaBrokerDown"
              expr  = "count(kafka_server_replicamanager_leadercount) < 3"
              for   = "5m"
              labels = {
                severity = "critical"
              }
              annotations = {
                summary     = "Kafka broker down"
                description = "Expected 3 Kafka brokers but only {{ $value }} are reporting."
              }
            },
          ]
        },
        {
          name = "clario360.infrastructure.alerts"
          rules = [
            {
              alert = "HighMemoryUsage"
              expr  = "container_memory_working_set_bytes{namespace=\"clario360\"} / container_spec_memory_limit_bytes{namespace=\"clario360\"} > 0.9"
              for   = "5m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "High memory usage: {{ $labels.pod }}"
                description = "Pod {{ $labels.pod }} is using >90% of its memory limit."
              }
            },
            {
              alert = "HighCPUUsage"
              expr  = "rate(container_cpu_usage_seconds_total{namespace=\"clario360\"}[5m]) / container_spec_cpu_quota{namespace=\"clario360\"} * 100000 > 0.9"
              for   = "10m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "High CPU usage: {{ $labels.pod }}"
                description = "Pod {{ $labels.pod }} is using >90% of its CPU limit for 10 minutes."
              }
            },
            {
              alert = "PVCAlmostFull"
              expr  = "kubelet_volume_stats_used_bytes{namespace=~\"clario360|kafka|monitoring\"} / kubelet_volume_stats_capacity_bytes > 0.85"
              for   = "5m"
              labels = {
                severity = "warning"
              }
              annotations = {
                summary     = "PVC almost full: {{ $labels.persistentvolumeclaim }}"
                description = "PVC {{ $labels.persistentvolumeclaim }} is {{ $value | humanizePercentage }} full."
              }
            },
          ]
        },
      ]
    }
  }

  depends_on = [helm_release.prometheus_stack]
}

# AlertManager configuration for Slack notifications
resource "kubernetes_secret" "alertmanager_config" {
  count = var.slack_webhook_url != "" ? 1 : 0

  metadata {
    name      = "alertmanager-slack-config"
    namespace = var.namespace
  }

  data = {
    "alertmanager.yaml" = yamlencode({
      global = {
        resolve_timeout = "5m"
      }
      route = {
        group_by        = ["alertname", "service"]
        group_wait      = "30s"
        group_interval  = "5m"
        repeat_interval = "4h"
        receiver        = "slack-notifications"
        routes = [
          {
            match    = { severity = "critical" }
            receiver = "slack-critical"
          },
        ]
      }
      receivers = [
        {
          name = "slack-notifications"
          slack_configs = [{
            api_url  = var.slack_webhook_url
            channel  = "#clario360-alerts"
            title    = "[{{ .Status | toUpper }}] {{ .CommonLabels.alertname }}"
            text     = "{{ range .Alerts }}{{ .Annotations.description }}\n{{ end }}"
            send_resolved = true
          }]
        },
        {
          name = "slack-critical"
          slack_configs = [{
            api_url  = var.slack_webhook_url
            channel  = "#clario360-critical"
            title    = "CRITICAL: {{ .CommonLabels.alertname }}"
            text     = "{{ range .Alerts }}{{ .Annotations.description }}\n{{ end }}"
            send_resolved = true
          }]
        },
      ]
    })
  }
}
