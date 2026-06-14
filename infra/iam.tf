# ────────────────────────────────────────────────────────────────────────────
# Project data source
# ────────────────────────────────────────────────────────────────────────────

data "google_project" "project" {
  project_id = var.project_id
}

# ────────────────────────────────────────────────────────────────────────────
# Service accounts
# ────────────────────────────────────────────────────────────────────────────

resource "google_service_account" "bastion" {
  account_id   = "sa-bastion"
  display_name = "Bastion VM service account"
  project      = var.project_id
}

resource "google_service_account" "ops" {
  account_id   = "sa-ops"
  display_name = "Ops VM service account"
  project      = var.project_id
}

resource "google_service_account" "gke_node" {
  account_id   = "sa-gke-node"
  display_name = "GKE node service account"
  project      = var.project_id
}

# ────────────────────────────────────────────────────────────────────────────
# Google service agents (locals)
# ────────────────────────────────────────────────────────────────────────────

locals {
  container_engine_robot_sa = "service-${data.google_project.project.number}@container-engine-robot.iam.gserviceaccount.com"
  gcs_service_agent_sa      = "service-${data.google_project.project.number}@gs-project-accounts.iam.gserviceaccount.com"
  compute_service_agent_sa  = "service-${data.google_project.project.number}@compute-system.iam.gserviceaccount.com"
}

# ────────────────────────────────────────────────────────────────────────────
# GKE node SA — minimal IAM roles
# ────────────────────────────────────────────────────────────────────────────

resource "google_project_iam_member" "gke_node_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.gke_node.email}"
}

resource "google_project_iam_member" "gke_node_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.gke_node.email}"
}

resource "google_project_iam_member" "gke_node_monitoring_viewer" {
  project = var.project_id
  role    = "roles/monitoring.viewer"
  member  = "serviceAccount:${google_service_account.gke_node.email}"
}

resource "google_project_iam_member" "gke_node_ar_reader" {
  project = var.project_id
  role    = "roles/artifactregistry.reader"
  member  = "serviceAccount:${google_service_account.gke_node.email}"
}

# ────────────────────────────────────────────────────────────────────────────
# Bastion SA — minimal IAM roles (W-3)
# ────────────────────────────────────────────────────────────────────────────

resource "google_project_iam_member" "bastion_log_writer" {
  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.bastion.email}"
}

resource "google_project_iam_member" "bastion_metric_writer" {
  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.bastion.email}"
}

# ────────────────────────────────────────────────────────────────────────────
# Ops SA — GCS backups bucket access
# ────────────────────────────────────────────────────────────────────────────

resource "google_storage_bucket_iam_member" "ops_gcs_writer" {
  bucket = google_storage_bucket.backups.name
  role   = "roles/storage.objectCreator"
  member = "serviceAccount:${google_service_account.ops.email}"
}

# ────────────────────────────────────────────────────────────────────────────
# KMS IAM — GKE etcd CMEK
# ────────────────────────────────────────────────────────────────────────────

resource "google_kms_crypto_key_iam_member" "gke_sa_encrypt_decrypt" {
  crypto_key_id = google_kms_crypto_key.gke.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${local.container_engine_robot_sa}"
}

# ────────────────────────────────────────────────────────────────────────────
# KMS IAM — GCS CMEK
# ────────────────────────────────────────────────────────────────────────────

resource "google_kms_crypto_key_iam_member" "gcs_sa_encrypt_decrypt" {
  crypto_key_id = google_kms_crypto_key.storage.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${local.gcs_service_agent_sa}"
}

# ────────────────────────────────────────────────────────────────────────────
# KMS IAM — node boot disk CMEK (W-4)
# ────────────────────────────────────────────────────────────────────────────

resource "google_kms_crypto_key_iam_member" "compute_sa_node_disk" {
  crypto_key_id = google_kms_crypto_key.node_disk.id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${local.compute_service_agent_sa}"
}
