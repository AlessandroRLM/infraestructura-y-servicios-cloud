locals {
  gcp_apis = [
    "compute.googleapis.com",
    "container.googleapis.com",
    "storage-api.googleapis.com",
    "cloudkms.googleapis.com",
    "logging.googleapis.com",
    "monitoring.googleapis.com",
    "iam.googleapis.com",
    "serviceusage.googleapis.com",
  ]
}

resource "google_project_service" "apis" {
  for_each = toset(local.gcp_apis)

  project            = var.project_id
  service            = each.value
  disable_on_destroy = false
}
