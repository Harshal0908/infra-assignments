terraform {
  required_version = ">= 1.8"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
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

output "namespace" {
  description = "Namespace used by the config service"
  value       = kubernetes_namespace_v1.config_service.metadata[0].name
}
