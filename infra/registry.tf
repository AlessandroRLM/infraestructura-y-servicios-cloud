resource "google_artifact_registry_repository" "academico" {
  location      = var.region
  repository_id = "academico"
  format        = "DOCKER"
  project       = var.project_id

  kms_key_name = google_kms_crypto_key.storage.id

  depends_on = [
    google_project_service.apis["artifactregistry.googleapis.com"],
    google_kms_crypto_key_iam_member.ar_sa_encrypt_decrypt,
  ]
}
