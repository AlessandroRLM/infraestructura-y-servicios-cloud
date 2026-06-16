provider "google" {
  project = var.project_id
  region  = var.region
  zone    = var.zone

  # google_billing_budget with user ADCs requires the provider to attribute
  # quota/billing to a project via the X-Goog-User-Project header; the provider
  # does not read the ADC quota_project_id on its own.
  billing_project       = var.project_id
  user_project_override = true
}

provider "aws" {
  region = var.aws_region
}

provider "random" {}
