resource "google_artifact_registry_repository" "academico" {
  location      = var.region
  repository_id = "academico"
  format        = "DOCKER"
  project       = var.project_id

  # Google-managed encryption at rest (no CMEK): forcing the AR service agent
  # for a customer key is a beta-only operation, avoided here. GKE etcd and GCS
  # backups keep CMEK; image storage uses Google-managed keys.
  depends_on = [
    google_project_service.apis["artifactregistry.googleapis.com"],
  ]
}
