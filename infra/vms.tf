resource "google_compute_address" "bastion" {
  name    = "bastion-ip"
  region  = var.region
  project = var.project_id
}

resource "google_compute_instance" "bastion" {
  name         = "bastion"
  machine_type = "e2-micro"
  zone         = var.zone
  project      = var.project_id

  tags = ["bastion"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.public.id

    access_config {
      nat_ip = google_compute_address.bastion.address
    }
  }

  metadata = {
    enable-oslogin = "TRUE"
  }

  # Minimal scopes: logging + monitoring only (W-2)
  service_account {
    email = google_service_account.bastion.email
    scopes = [
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
    ]
  }
}

resource "google_compute_instance" "ops" {
  name         = "ops"
  machine_type = "e2-small"
  zone         = var.zone
  project      = var.project_id

  tags = ["ops"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.ops.id
  }

  metadata = {
    enable-oslogin = "TRUE"
    startup-script = templatefile("${path.module}/scripts/ops-startup.sh.tftpl", {
      backup_script  = file("${path.module}/scripts/backup.sh")
      restore_script = file("${path.module}/scripts/restore.sh")
      gcs_bucket     = google_storage_bucket.backups.name
      s3_bucket      = aws_s3_bucket.backups_dr.bucket
      cluster_name   = google_container_cluster.primary.name
      zone           = var.zone
      namespace      = "prod"
    })
  }

  # Minimal scopes: storage read/write + logging + monitoring (W-2)
  service_account {
    email = google_service_account.ops.email
    scopes = [
      "https://www.googleapis.com/auth/devstorage.read_write",
      "https://www.googleapis.com/auth/logging.write",
      "https://www.googleapis.com/auth/monitoring.write",
    ]
  }
}

# Attach daily snapshot policy to bastion and ops boot disks (S-2)
# Postgres PVC disk snapshots are handled by the GKE StorageClass — out of scope here.
resource "google_compute_disk_resource_policy_attachment" "bastion_snapshot" {
  name    = google_compute_resource_policy.daily_snapshot.name
  disk    = google_compute_instance.bastion.name
  zone    = var.zone
  project = var.project_id
}

resource "google_compute_disk_resource_policy_attachment" "ops_snapshot" {
  name    = google_compute_resource_policy.daily_snapshot.name
  disk    = google_compute_instance.ops.name
  zone    = var.zone
  project = var.project_id
}
