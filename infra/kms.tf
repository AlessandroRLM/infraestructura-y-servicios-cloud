resource "google_kms_key_ring" "academico" {
  name     = "academico"
  location = var.region
  project  = var.project_id

  depends_on = [google_project_service.apis["cloudkms.googleapis.com"]]
}

resource "google_kms_crypto_key" "gke" {
  name            = "gke-etcd"
  key_ring        = google_kms_key_ring.academico.id
  rotation_period = "7776000s"

  lifecycle {
    prevent_destroy = true
  }
}

resource "google_kms_crypto_key" "storage" {
  name            = "storage"
  key_ring        = google_kms_key_ring.academico.id
  rotation_period = "7776000s"

  lifecycle {
    prevent_destroy = true
  }
}

# KMS key for GKE node boot disk CMEK (W-4)
resource "google_kms_crypto_key" "node_disk" {
  name            = "node-disk"
  key_ring        = google_kms_key_ring.academico.id
  rotation_period = "7776000s"

  lifecycle {
    prevent_destroy = true
  }
}
