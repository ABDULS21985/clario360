# =============================================================================
# Kubernetes Network Policies
# Enforce microsegmentation between services — deny by default, allow explicitly
# =============================================================================

# Default deny all ingress in clario360 namespace
resource "kubernetes_network_policy" "default_deny" {
  metadata {
    name      = "default-deny-all"
    namespace = "clario360"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress", "Egress"]

    # Deny all ingress by default
    ingress = []

    # Allow DNS egress + internal namespace egress
    egress {
      to {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "kube-system"
          }
        }
      }
      ports {
        port     = "53"
        protocol = "UDP"
      }
      ports {
        port     = "53"
        protocol = "TCP"
      }
    }
  }
}

# Allow API Gateway to receive ingress from ingress-nginx
resource "kubernetes_network_policy" "allow_ingress_to_gateway" {
  metadata {
    name      = "allow-ingress-to-gateway"
    namespace = "clario360"
  }

  spec {
    pod_selector {
      match_labels = {
        "app.kubernetes.io/name" = "api-gateway"
      }
    }

    policy_types = ["Ingress"]

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "ingress-nginx"
          }
        }
      }
      ports {
        port     = "8080"
        protocol = "TCP"
      }
    }
  }
}

# Allow internal service-to-service communication
resource "kubernetes_network_policy" "allow_internal" {
  metadata {
    name      = "allow-clario360-internal"
    namespace = "clario360"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress", "Egress"]

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "clario360"
          }
        }
      }
      ports {
        port     = "8080"
        protocol = "TCP"
      }
    }

    egress {
      to {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "clario360"
          }
        }
      }
    }
  }
}

# Allow services to reach Kafka
resource "kubernetes_network_policy" "allow_kafka_egress" {
  metadata {
    name      = "allow-kafka-egress"
    namespace = "clario360"
  }

  spec {
    pod_selector {}

    policy_types = ["Egress"]

    egress {
      to {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "kafka"
          }
        }
      }
      ports {
        port     = "9092"
        protocol = "TCP"
      }
      ports {
        port     = "9093"
        protocol = "TCP"
      }
    }
  }
}

# Allow services to reach monitoring (metrics endpoint)
resource "kubernetes_network_policy" "allow_monitoring_egress" {
  metadata {
    name      = "allow-monitoring-scrape"
    namespace = "clario360"
  }

  spec {
    pod_selector {}

    policy_types = ["Ingress"]

    ingress {
      from {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "monitoring"
          }
        }
      }
      ports {
        port     = "8080"
        protocol = "TCP"
      }
      ports {
        port     = "9090"
        protocol = "TCP"
      }
    }
  }
}

# Allow services to reach Vault
resource "kubernetes_network_policy" "allow_vault_egress" {
  metadata {
    name      = "allow-vault-egress"
    namespace = "clario360"
  }

  spec {
    pod_selector {}

    policy_types = ["Egress"]

    egress {
      to {
        namespace_selector {
          match_labels = {
            "kubernetes.io/metadata.name" = "vault"
          }
        }
      }
      ports {
        port     = "8200"
        protocol = "TCP"
      }
    }
  }
}
