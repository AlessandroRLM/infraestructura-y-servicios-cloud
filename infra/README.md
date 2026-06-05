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
├── backend.tf      # estado remoto (GCS)
├── providers.tf    # google + aws
├── variables.tf
├── network.tf      # VPC, subredes, NAT, firewall
├── vms.tf          # bastion + ops
├── gke.tf          # cluster + node pool
├── storage.tf      # buckets GCS + S3
├── backup.tf       # IAM AWS + cron de backup
└── outputs.tf
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
