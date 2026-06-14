variable "project_id" {
  type        = string
  description = "GCP project ID."
}

variable "region" {
  type        = string
  description = "GCP region."
  default     = "us-central1"
}

variable "zone" {
  type        = string
  description = "GCP zone (zonal GKE cluster)."
  default     = "us-central1-a"
}

variable "aws_region" {
  type        = string
  description = "AWS region for DR S3 bucket."
  default     = "us-east-1"
}

variable "admin_ip" {
  type        = string
  description = "Admin CIDR (e.g. 203.0.113.0/32) for SSH firewall and GKE master authorized networks. Required; must be a valid CIDR block."

  validation {
    condition     = can(cidrnetmask(var.admin_ip))
    error_message = "admin_ip must be a valid CIDR block (e.g. 203.0.113.0/32)."
  }
}

variable "node_machine_type" {
  type        = string
  description = "GKE node machine type."
  default     = "e2-medium"
}

variable "node_min_count" {
  type        = number
  description = "GKE node pool minimum node count."
  default     = 2
}

variable "node_max_count" {
  type        = number
  description = "GKE node pool maximum node count."
  default     = 4
}

variable "alert_email" {
  type        = string
  description = "Email address for monitoring alert notifications."
}

variable "app_host" {
  type        = string
  description = "Public hostname of the application for uptime checks (e.g. academico.example.com)."
  default     = "academico.example.com"
}

variable "billing_account_id" {
  type        = string
  description = "GCP billing account ID for budget alerts (format: XXXXXX-XXXXXX-XXXXXX)."
}

variable "monthly_budget_usd" {
  type        = number
  description = "Monthly spend budget in USD; triggers alerts at 50%, 90%, and 100%."
  default     = 150

  validation {
    condition     = var.monthly_budget_usd == floor(var.monthly_budget_usd)
    error_message = "monthly_budget_usd must be a whole number."
  }
}
