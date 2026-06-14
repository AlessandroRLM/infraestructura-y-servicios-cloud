# Runbook — Despliegue real en GCP + AWS (copy-paste)

Guía lineal, comando por comando, para llevar el sistema a la nube. Ejecutar los bloques **en orden**. El diseño y el "por qué" están en [`../arquitectura`](../arquitectura/README.md) e [`../infraestructura`](../infraestructura/README.md); acá solo van los comandos.

> Para correr todo en local sin nube, ver [`../local-dev`](../local-dev/README.md).

Tres cosas a saber antes de empezar:
1. El cluster GKE tiene **endpoint privado** (seguridad): `kubectl` se ejecuta **desde la VM bastión**, no desde la máquina local (Fase B).
2. El usuario de GCP necesita los roles `roles/owner` (o `roles/editor` + `roles/resourcemanager.projectIamAdmin` + `roles/container.admin`) sobre el proyecto.
3. **Ejecutar todo desde una sola máquina con IP pública estable.** Esa IP va en `admin_ip` y habilita el SSH al bastión y el acceso al API server de GKE. Si la IP cambia (otra red, otra PC), actualizar `admin_ip` en `terraform.tfvars` y ejecutar `terraform apply` de nuevo.

---

## Fase 0 — Instalar las herramientas (en la máquina local)

Se requiere: `gcloud`, `terraform`, `docker`, `kubectl` (trae `kustomize`), `aws`, `git`, `openssl`. Seleccionar según el sistema operativo.

### Debian / Ubuntu

```bash
sudo apt-get update && sudo apt-get install -y curl git unzip openssl apt-transport-https ca-certificates gnupg

# Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker "$USER"   # cerrar la sesión y volver a entrar para usar docker sin sudo

# Google Cloud CLI (incluye gke-gcloud-auth-plugin)
echo "deb https://packages.cloud.google.com/apt cloud-sdk main" | sudo tee /etc/apt/sources.list.d/google-cloud-sdk.list
curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo gpg --dearmor -o /usr/share/keyrings/cloud.google.gpg
sudo apt-get update && sudo apt-get install -y google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin kubectl

# Terraform
wget -O- https://apt.releases.hashicorp.com/gpg | sudo gpg --dearmor -o /usr/share/keyrings/hashicorp-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/hashicorp-archive-keyring.gpg] https://apt.releases.hashicorp.com $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/hashicorp.list
sudo apt-get update && sudo apt-get install -y terraform

# AWS CLI v2
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o awscliv2.zip && unzip -q awscliv2.zip && sudo ./aws/install && rm -rf aws awscliv2.zip
```

### Fedora / RHEL

```bash
sudo dnf install -y curl git unzip openssl
curl -fsSL https://get.docker.com | sh && sudo usermod -aG docker "$USER"
# Google Cloud CLI
sudo tee /etc/yum.repos.d/google-cloud-sdk.repo <<EOF
[google-cloud-cli]
name=Google Cloud CLI
baseurl=https://packages.cloud.google.com/yum/repos/cloud-sdk-el9-x86_64
enabled=1
gpgcheck=1
repo_gpgcheck=0
gpgkey=https://packages.cloud.google.com/yum/doc/rpm-package-key.gpg
EOF
sudo dnf install -y google-cloud-cli google-cloud-cli-gke-gcloud-auth-plugin kubectl
# Terraform
sudo dnf install -y dnf-plugins-core
sudo dnf config-manager addrepo --from-repofile=https://rpm.releases.hashicorp.com/fedora/hashicorp.repo
sudo dnf install -y terraform
# AWS CLI v2
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o awscliv2.zip && unzip -q awscliv2.zip && sudo ./aws/install && rm -rf aws awscliv2.zip
```

### macOS (Homebrew)

```bash
brew install --cask docker        # abrir Docker Desktop una vez
brew install google-cloud-sdk terraform kubectl awscli git
gcloud components install gke-gcloud-auth-plugin
```

Verificar:

```bash
gcloud version && terraform version && docker --version && kubectl version --client && aws --version
```

---

## Fase 1 — Variables (completar una vez y pegar el bloque)

```bash
export PROJECT_ID="mi-proyecto-gcp"                 # project id de GCP
export REGION="us-central1"
export ZONE="us-central1-a"
export ADMIN_IP="$(curl -s ifconfig.me)/32"         # IP pública propia /32 (SSH + API server)
export DOMAIN="academico.midominio.com"             # dominio real de la app
export ALERT_EMAIL="ops@midominio.com"
export BILLING_ACCOUNT_ID="XXXXXX-XXXXXX-XXXXXX"     # gcloud billing accounts list
export AWS_REGION="us-east-1"
export REPO_URL="https://github.com/AlessandroRLM/infraestructura-y-servicios-cloud.git"
```

---

## Fase A — Desde la máquina local

### 1. Autenticación

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project "$PROJECT_ID"
aws configure                                        # claves del usuario IAM de backups
```

### 2. Vincular facturación

```bash
gcloud billing projects link "$PROJECT_ID" --billing-account="$BILLING_ACCOUNT_ID"
```

### 3. Bucket de estado de Terraform (una sola vez)

```bash
gcloud storage buckets create "gs://tfstate-academico" \
  --project="$PROJECT_ID" --location="$REGION" --uniform-bucket-level-access
gcloud storage buckets update "gs://tfstate-academico" --versioning
```

> Si el nombre `tfstate-academico` ya está tomado (los buckets son globales), elegir otro y actualizar `bucket = "..."` en `infra/backend.tf`.

### 4. Variables de Terraform

```bash
cd infra
cat > terraform.tfvars <<EOF
project_id         = "$PROJECT_ID"
admin_ip           = "$ADMIN_IP"
alert_email        = "$ALERT_EMAIL"
billing_account_id = "$BILLING_ACCOUNT_ID"
app_host           = "$DOMAIN"
aws_region         = "$AWS_REGION"
EOF
```

> `terraform.tfvars` está gitignored (no se commitea). `region`, `zone`, tamaños y budget usan defaults.

### 5. Aplicar la infraestructura

```bash
terraform init
terraform plan      # revisar el plan
terraform apply     # escribir 'yes' para confirmar
```

### 6. Guardar los outputs

```bash
export AR_URL=$(terraform output -raw artifact_registry_url)
export CLUSTER=$(terraform output -raw cluster_name)
echo "Registry: $AR_URL"
echo "Cluster:  $CLUSTER"
cd ..
```

### 7. Construir y subir las imágenes al Artifact Registry

```bash
gcloud auth configure-docker "${REGION}-docker.pkg.dev"

docker build --provenance=false -t "$AR_URL/api:1.0.0" backend/
docker build --provenance=false -f frontend/Dockerfile -t "$AR_URL/web:1.0.0" .

docker push "$AR_URL/api:1.0.0"
docker push "$AR_URL/web:1.0.0"
```

### 8. Apuntar el overlay prod a esas imágenes

```bash
cd k8s/overlays/prod
kustomize edit set image "academico/api=$AR_URL/api:1.0.0"
kustomize edit set image "academico/web=$AR_URL/web:1.0.0"
cd ../../..
```

---

## Fase B — Desde la VM bastión (el cluster es privado)

### 9. Entrar al bastión

```bash
gcloud compute ssh bastion --zone "$ZONE" --project "$PROJECT_ID"
```

### 10. Preparar el bastión (una sola vez, dentro de la sesión SSH)

**Primero, volver a pegar el bloque de variables de la Fase 1** (el bastión es otra shell y no las tiene). Luego:

```bash
sudo apt-get update
sudo apt-get install -y git kubectl google-cloud-cli-gke-gcloud-auth-plugin
git clone "$REPO_URL" && cd infraestructura-y-servicios-cloud
gcloud container clusters get-credentials gke-academico --zone "$ZONE"
```

### 11. Crear el Secret de producción (`app-secrets`)

```bash
kubectl create namespace academico-prod --dry-run=client -o yaml | kubectl apply -f -

kubectl -n academico-prod create secret generic app-secrets \
  --from-literal=DATABASE_URL='postgres://app:CAMBIA_ESTA_CLAVE@postgres:5432/academico?sslmode=disable' \
  --from-literal=POSTGRES_USER='app' \
  --from-literal=POSTGRES_PASSWORD='CAMBIA_ESTA_CLAVE' \
  --from-literal=METRICS_AUTH_TOKEN="$(openssl rand -hex 24)"
```

> En producción endurecida esto se inyecta desde Secret Manager con External Secrets; acá se crea a mano para arrancar. Usar la MISMA clave en `DATABASE_URL` y `POSTGRES_PASSWORD`.

### 12. Desplegar la aplicación

```bash
kubectl apply -k k8s/overlays/prod
kubectl -n academico-prod rollout status deploy/api
kubectl -n academico-prod rollout status deploy/web
kubectl -n academico-prod get pods
```

---

## Fase C — TLS y DNS

### 13. cert-manager + certificado (Let's Encrypt)

```bash
# instalar cert-manager (desde el bastión)
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
kubectl -n cert-manager rollout status deploy/cert-manager-webhook

# ClusterIssuer
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: ${ALERT_EMAIL}
    privateKeySecretRef:
      name: letsencrypt-account
    solvers:
      - http01:
          ingress:
            ingressClassName: nginx
EOF

# anotar el Ingress para que cert-manager emita el cert academico-prod-tls
kubectl -n academico-prod annotate ingress academico \
  cert-manager.io/cluster-issuer=letsencrypt --overwrite
```

### 14. DNS

```bash
# IP pública del Ingress (esperar 1-2 min a que se asigne)
kubectl -n academico-prod get ingress academico -o jsonpath='{.status.loadBalancer.ingress[0].ip}'; echo
```

Crear un registro **A** del dominio (`$DOMAIN`) apuntando a esa IP. Cuando el DNS propague, cert-manager emite el certificado y la app queda en `https://$DOMAIN`.

---

## Fase D — Verificación

```bash
# pods de prod
kubectl -n academico-prod get pods,svc,ingress

# certificado emitido
kubectl -n academico-prod get certificate

# app por HTTPS (cuando DNS + cert estén listos)
curl -s -o /dev/null -w "%{http_code}\n" "https://$DOMAIN/"
```

Dashboards y alertas: consola GCP → Monitoring → Dashboards (`infra`, `app`, `costos`) y Alerting (CPU, uptime, RPC errors) + Budgets.

---

## Fase E — Backups (verificación)

```bash
# ejecutar el backup manualmente desde la VM ops y verificar en ambas nubes
gcloud compute ssh ops --zone "$ZONE" --project "$PROJECT_ID" \
  --command 'sudo bash -c ". /etc/default/academico-backup && /opt/backup/backup.sh"'

gcloud storage ls "gs://$(cd infra && terraform output -raw gcs_backups_bucket)/"
aws s3 ls "s3://$(cd infra && terraform output -raw s3_backups_dr_bucket)/"
```

> La VM `ops` necesita las credenciales AWS configuradas (`aws configure`) — se proveen fuera de banda (no viven en el state). La prueba de restauración está en [`../infraestructura`](../infraestructura/README.md) §10 (`infra/scripts/restore.sh`).

---

## Fase F — Apagado (ahorro de costos)

```bash
cd infra
terraform destroy   # elimina toda la infra; el bucket de tfstate persiste
```

> El estado en GCS sobrevive, así que `terraform apply` reconstruye todo idéntico al retomar. Ver optimización de costos en [`../monitoreo-costos`](../monitoreo-costos/README.md).
