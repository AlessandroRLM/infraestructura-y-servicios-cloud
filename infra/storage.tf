resource "random_id" "bucket_suffix" {
  byte_length = 4
}

locals {
  bucket_suffix = random_id.bucket_suffix.hex
}

resource "google_storage_bucket" "assets" {
  name                        = "assets-academico-${local.bucket_suffix}"
  location                    = var.region
  project                     = var.project_id
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  encryption {
    default_kms_key_name = google_kms_crypto_key.storage.id
  }

  depends_on = [google_kms_crypto_key_iam_member.gcs_sa_encrypt_decrypt]
}

resource "google_storage_bucket" "backups" {
  name                        = "backups-academico-${local.bucket_suffix}"
  location                    = var.region
  project                     = var.project_id
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  lifecycle_rule {
    action {
      type = "Delete"
    }
    condition {
      age = 30
    }
  }

  encryption {
    default_kms_key_name = google_kms_crypto_key.storage.id
  }

  depends_on = [google_kms_crypto_key_iam_member.gcs_sa_encrypt_decrypt]

  lifecycle {
    prevent_destroy = true
  }
}

# tfstate bucket is bootstrapped out-of-band (see docs/infraestructura §2);
# it cannot be managed by the state it stores. (W-6)

resource "aws_s3_bucket" "backups_dr" {
  bucket = "backups-academico-dr-${local.bucket_suffix}"
}

resource "aws_s3_bucket_versioning" "backups_dr" {
  bucket = aws_s3_bucket.backups_dr.id

  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "backups_dr" {
  bucket = aws_s3_bucket.backups_dr.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_public_access_block" "backups_dr" {
  bucket = aws_s3_bucket.backups_dr.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_lifecycle_configuration" "backups_dr" {
  bucket = aws_s3_bucket.backups_dr.id

  rule {
    id     = "transition-to-glacier"
    status = "Enabled"

    filter {}

    transition {
      days          = 30
      storage_class = "GLACIER"
    }
  }
}

resource "aws_iam_user" "ops_backup" {
  name = "ops-backup"
}

data "aws_iam_policy_document" "ops_backup" {
  statement {
    effect = "Allow"
    actions = [
      "s3:PutObject",
      "s3:GetObject",
    ]
    resources = ["${aws_s3_bucket.backups_dr.arn}/*"]
  }

  statement {
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [aws_s3_bucket.backups_dr.arn]
  }
}

resource "aws_iam_user_policy" "ops_backup" {
  name   = "ops-backup-s3-policy"
  user   = aws_iam_user.ops_backup.name
  policy = data.aws_iam_policy_document.ops_backup.json
}

# Access key is provisioned out-of-band and stored in Secret Manager / injected to the ops VM.
# Long-lived keys must not be managed in state. (C-1)
