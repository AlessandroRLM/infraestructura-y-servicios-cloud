# tfstate-academico is created out-of-band (see docs/infraestructura §2) — not managed here.
terraform {
  backend "gcs" {
    bucket = "tfstate-academico"
    prefix = "infra"
  }
}
