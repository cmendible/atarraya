terraform {
  required_version = "> 0.12"

  required_providers {
    azurerm = ">=1.38.0"
    azuread = "~> 0.7"
  }
}

resource "azurerm_resource_group" "rg" {
  name     = var.resource_group_name
  location = var.location
}

resource "azurerm_kubernetes_cluster" "k8s" {
  name                = var.cluster_name
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  dns_prefix          = var.dns_prefix

  default_node_pool {
    name       = "default"
    node_count = var.agent_count
    vm_size    = "Standard_D2_v2"
  }

  service_principal {
    client_id     = var.client_id
    client_secret = var.client_secret
  }

  role_based_access_control {
    enabled = true
  }
}

data "azurerm_subscription" "current" {}

resource "azurerm_key_vault" "kv" {
  name                = var.key_vault_name
  location            = azurerm_resource_group.rg.location
  resource_group_name = azurerm_resource_group.rg.name
  tenant_id           = data.azurerm_subscription.current.tenant_id

  sku_name = "standard"

  access_policy {
    tenant_id = data.azurerm_subscription.current.tenant_id
    object_id = var.managed_identity_client_id

    key_permissions = []

    secret_permissions = [
      "get",
      "list"
    ]

    storage_permissions = []
  }
}

resource "azurerm_key_vault_secret" "secret" {
  name         = "secret"
  value        = "SUPER SECRET"
  key_vault_id = azurerm_key_vault.kv.id
}

# provider "kubernetes" {
#   host                   = "${azurerm_kubernetes_cluster.k8s.kube_config.0.host}"
#   client_certificate     = "${base64decode(azurerm_kubernetes_cluster.k8s.kube_config.0.client_certificate)}"
#   client_key             = "${base64decode(azurerm_kubernetes_cluster.k8s.kube_config.0.client_key)}"
#   cluster_ca_certificate = "${base64decode(azurerm_kubernetes_cluster.k8s.kube_config.0.cluster_ca_certificate)}"
#   alias                  = "k8s"
# }

