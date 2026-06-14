# Infra — IaC

Infraestructura como código del proyecto: GCP (principal) y AWS (respaldo), con Terraform. Toda la infraestructura se crea con `terraform apply` y se elimina con `terraform destroy`.

La guía de despliegue paso a paso está en [`docs/infraestructura`](../docs/infraestructura/README.md).

## Qué crea

| Recurso | Detalle |
|---------|---------|
| Red | VPC `vpc-academico`, subredes (pública para bastión, privadas para cargas), Cloud NAT, firewall. |
| VMs | `bastion` (acceso SSH/kubectl) y `ops` (cron de backup cross-cloud). |
| Cluster | GKE zonal + node pool. |
| Storage | Buckets GCS (activos, backups, state) y bucket S3 de respaldo. |
| Backup | IAM de AWS y cron en la VM `ops` (GCS → S3). |

## Estructura

```
infra/
├── versions.tf                 # versiones de Terraform y providers (google + aws + random)
├── providers.tf                # configuración de providers
├── backend.tf                  # estado remoto (GCS)
├── variables.tf
├── apis.tf                     # habilitación de APIs de GCP
├── network.tf                  # VPC, subredes, Cloud NAT, firewall
├── kms.tf                      # key ring + claves CMEK (cifrado en reposo)
├── iam.tf                      # service accounts + service agents + bindings IAM
├── vms.tf                      # bastion + ops (startup-script del cron de backup)
├── gke.tf                      # cluster + node pool
├── registry.tf                 # Artifact Registry (Docker, CMEK)
├── storage.tf                  # buckets GCS + bucket S3 (DR) + IAM AWS
├── snapshots.tf                # política de snapshot diario (disco postgres)
├── monitoring.tf               # dashboards, alertas, uptime check, budget
├── outputs.tf
├── terraform.tfvars.example
└── scripts/                    # backup.sh, restore.sh, ops-startup.sh.tftpl
```

## Uso

Requisitos: Terraform >= 1.6, `gcloud` y `aws` autenticados.

```bash
terraform init      # conecta el backend GCS y descarga providers
terraform plan      # revisar antes de aplicar
terraform apply     # crear la infraestructura

terraform destroy   # eliminar todo (ahorro fuera de ventanas de uso)
```

El estado se guarda en un bucket GCS con versionado, así que `apply` reconstruye la infraestructura idéntica al retomar.
