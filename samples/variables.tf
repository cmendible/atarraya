variable "client_id" {}
variable "client_secret" {}

variable resource_group_name {
  default = "atarraya-sample"
}

variable location {
  default = "West Europe"
}

variable cluster_name {
  default = "atarraya-sample"
}

variable "dns_prefix" {
  default = "atarraya-sample"
}

variable "agent_count" {
  default = 3
}

variable key_vault_name {
  default = "atarraya-vault"
}

variable managed_identity_client_id {}
