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

output "namespace" {
  description = "Namespace used by the config service"
  value       = kubernetes_namespace_v1.config_service.metadata[0].name
}