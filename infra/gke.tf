resource "google_container_cluster" "primary" {
  provider = google

  name     = "gke-academico"
  location = var.zone
  project  = var.project_id

  # deletion_protection = false: infra is torn down off-hours by design (cost schedule)
  deletion_protection = false

  network    = google_compute_network.vpc.id
  subnetwork = google_compute_subnetwork.gke.id

  remove_default_node_pool = true
  initial_node_count       = 1

  ip_allocation_policy {
    cluster_secondary_range_name  = "pods"
    services_secondary_range_name = "services"
  }

  datapath_provider = "ADVANCED_DATAPATH"

  private_cluster_config {
    enable_private_nodes    = true
    enable_private_endpoint = true
    master_ipv4_cidr_block  = "172.16.0.0/28"
  }

  # API server is only reachable from the bastion subnet and the admin CIDR (C-2)
  master_authorized_networks_config {
    cidr_blocks {
      cidr_block   = "10.0.0.0/24"
      display_name = "bastion-subnet"
    }

    cidr_blocks {
      cidr_block   = var.admin_ip
      display_name = "admin"
    }
  }

  workload_identity_config {
    workload_pool = "${var.project_id}.svc.id.goog"
  }

  release_channel {
    channel = "REGULAR"
  }

  database_encryption {
    state    = "ENCRYPTED"
    key_name = google_kms_crypto_key.gke.id
  }

  logging_config {
    enable_components = ["SYSTEM_COMPONENTS", "WORKLOADS"]
  }

  monitoring_config {
    enable_components = ["SYSTEM_COMPONENTS"]
  }

  depends_on = [
    google_project_service.apis["container.googleapis.com"],
    google_kms_crypto_key_iam_member.gke_sa_encrypt_decrypt,
  ]
}

resource "google_container_node_pool" "primary" {
  name     = "primary-pool"
  location = var.zone
  cluster  = google_container_cluster.primary.name
  project  = var.project_id

  autoscaling {
    min_node_count = var.node_min_count
    max_node_count = var.node_max_count
  }

  node_config {
    machine_type = var.node_machine_type
    disk_size_gb = 50
    image_type   = "COS_CONTAINERD"

    # Node boot disk CMEK (W-4)
    boot_disk_kms_key = google_kms_crypto_key.node_disk.id

    service_account = google_service_account.gke_node.email

    # cloud-platform scope is the GKE-recommended pattern for node pools;
    # least-privilege is enforced via the node SA's IAM bindings in iam.tf (W-2)
    oauth_scopes = ["https://www.googleapis.com/auth/cloud-platform"]

    shielded_instance_config {
      enable_secure_boot          = true
      enable_integrity_monitoring = true
    }

    workload_metadata_config {
      mode = "GKE_METADATA"
    }

    tags = ["gke-node"]
  }

  depends_on = [google_kms_crypto_key_iam_member.compute_sa_node_disk]
}
