output "cluster_name" {
  description = "GKE cluster name."
  value       = google_container_cluster.primary.name
}

output "cluster_endpoint" {
  description = "GKE cluster endpoint."
  value       = google_container_cluster.primary.endpoint
  sensitive   = true
}

output "bastion_public_ip" {
  description = "Bastion VM public IP address."
  value       = google_compute_address.bastion.address
}

output "gcs_assets_bucket" {
  description = "GCS assets bucket name."
  value       = google_storage_bucket.assets.name
}

output "gcs_backups_bucket" {
  description = "GCS backups bucket name."
  value       = google_storage_bucket.backups.name
}

output "s3_backups_dr_bucket" {
  description = "AWS S3 DR backups bucket name."
  value       = aws_s3_bucket.backups_dr.id
}

output "kms_gke_key_id" {
  description = "KMS crypto key ID for GKE etcd encryption."
  value       = google_kms_crypto_key.gke.id
}

output "kms_storage_key_id" {
  description = "KMS crypto key ID for GCS CMEK."
  value       = google_kms_crypto_key.storage.id
}

output "artifact_registry_url" {
  description = "Artifact Registry base URL for Docker images (region-docker.pkg.dev/project/repo)."
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${google_artifact_registry_repository.academico.repository_id}"
}
