terraform {
  required_version = ">= 1.8"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 3.0"
    }

    random = {
      source  = "hashicorp/random"
      version = "~> 3.0"
    }
  }
}

provider "kubernetes" {
  config_path    = "~/.kube/config"
  config_context = "kind-config-service"
}

variable "namespace" {
  description = "Kubernetes namespace for the config service"
  type        = string
  default     = "config-service"
}

locals {
  app_name = "config-service"
  db_name  = "config_service"
  db_user  = "config_service"

  common_labels = {
    app        = local.app_name
    managed-by = "terraform"
  }

  postgres_labels = {
    app        = "postgres"
    component  = "database"
    managed-by = "terraform"
  }
}

resource "kubernetes_namespace_v1" "config_service" {
  metadata {
    name   = var.namespace
    labels = local.common_labels
  }
}

resource "random_password" "database" {
  length  = 24
  special = false
}

resource "kubernetes_config_map_v1" "config_service" {
  metadata {
    name      = local.app_name
    namespace = kubernetes_namespace_v1.config_service.metadata[0].name
    labels    = local.common_labels
  }

  data = {
    APP_PORT  = "8080"
    LOG_LEVEL = "INFO"
  }
}

resource "kubernetes_secret_v1" "database" {
  metadata {
    name      = "config-service-db"
    namespace = kubernetes_namespace_v1.config_service.metadata[0].name
    labels    = local.common_labels
  }

  type = "Opaque"

  data = {
    DB_USER     = local.db_user
    DB_PASSWORD = random_password.database.result

    DATABASE_URL = format(
      "postgres://%s:%s@postgres:5432/%s?sslmode=disable",
      local.db_user,
      random_password.database.result,
      local.db_name
    )
  }
}

resource "kubernetes_persistent_volume_claim_v1" "postgres" {
  wait_until_bound = false

  metadata {
    name      = "postgres-data"
    namespace = kubernetes_namespace_v1.config_service.metadata[0].name
    labels    = local.postgres_labels
  }

  spec {
    access_modes = ["ReadWriteOnce"]

    resources {
      requests = {
        storage = "1Gi"
      }
    }
  }
}

resource "kubernetes_service_v1" "postgres" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace_v1.config_service.metadata[0].name
    labels    = local.postgres_labels
  }

  spec {
    selector = {
      app = "postgres"
    }

    port {
      name        = "postgres"
      port        = 5432
      target_port = 5432
      protocol    = "TCP"
    }

    cluster_ip = "None"
  }
}

resource "kubernetes_stateful_set_v1" "postgres" {
  metadata {
    name      = "postgres"
    namespace = kubernetes_namespace_v1.config_service.metadata[0].name
    labels    = local.postgres_labels
  }

  spec {
    service_name = kubernetes_service_v1.postgres.metadata[0].name
    replicas     = 1

    selector {
      match_labels = {
        app = "postgres"
      }
    }

    template {
      metadata {
        labels = {
          app = "postgres"
        }
      }

      spec {
        container {
          name  = "postgres"
          image = "postgres:16-alpine"

          port {
            name           = "postgres"
            container_port = 5432
          }

          env {
            name  = "POSTGRES_DB"
            value = local.db_name
          }

          env {
            name  = "PGDATA"
            value = "/var/lib/postgresql/data/pgdata"
          }

          env {
            name = "POSTGRES_USER"

            value_from {
              secret_key_ref {
                name = kubernetes_secret_v1.database.metadata[0].name
                key  = "DB_USER"
              }
            }
          }

          env {
            name = "POSTGRES_PASSWORD"

            value_from {
              secret_key_ref {
                name = kubernetes_secret_v1.database.metadata[0].name
                key  = "DB_PASSWORD"
              }
            }
          }

          readiness_probe {
            exec {
              command = [
                "sh",
                "-c",
                "pg_isready -U \"$POSTGRES_USER\" -d \"$POSTGRES_DB\""
              ]
            }

            initial_delay_seconds = 5
            period_seconds        = 5
            timeout_seconds       = 3
            failure_threshold     = 6
          }

          resources {
            requests = {
              cpu    = "100m"
              memory = "128Mi"
            }

            limits = {
              cpu    = "500m"
              memory = "512Mi"
            }
          }

          volume_mount {
            name       = "postgres-data"
            mount_path = "/var/lib/postgresql/data"
          }
        }

        volume {
          name = "postgres-data"

          persistent_volume_claim {
            claim_name = kubernetes_persistent_volume_claim_v1.postgres.metadata[0].name
          }
        }
      }
    }
  }
}

output "namespace" {
  description = "Namespace used by the config service"
  value       = kubernetes_namespace_v1.config_service.metadata[0].name
}