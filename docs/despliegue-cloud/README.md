# Runbook — Despliegue real en GCP + AWS (copy-paste)

Guía lineal, comando por comando, para llevar el sistema a la nube. Ejecutar los bloques **en orden**. El diseño y el "por qué" están en [`../arquitectura`](../arquitectura/README.md) e [`../infraestructura`](../infraestructura/README.md); acá solo van los comandos.

> Para correr todo en local sin nube, ver [`../despliegue-local`](../despliegue-local/README.md).

Tres cosas a saber antes de empezar:
1. El cluster GKE tiene **endpoint privado** (seguridad): `kubectl` se ejecuta **desde la VM bastión**, no desde la máquina local (§4).
2. El usuario de GCP necesita los roles `roles/owner` (o `roles/editor` + `roles/resourcemanager.projectIamAdmin` + `roles/container.admin`) sobre el proyecto.
3. **Ejecutar todo desde una sola máquina con IP pública IPv4 estable.** Esa IP va en `admin_ip` y habilita el SSH al bastión (el acceso al API server de GKE es solo desde el bastión por endpoint privado, no desde `admin_ip`). Si la IP cambia (otra red, otra PC), actualizar `admin_ip` en `terraform.tfvars` y ejecutar `terraform apply` de nuevo. `admin_ip` debe ser IPv4 con `/32`; IPv6 no es válida.

---

## Índice

1. [Instalar las herramientas (en la máquina local)](#1-instalar-las-herramientas-en-la-máquina-local)
2. [Variables (completar una vez y pegar el bloque)](#2-variables-completar-una-vez-y-pegar-el-bloque)
3. [Desde la máquina local: infraestructura e imágenes](#3-desde-la-máquina-local-infraestructura-e-imágenes)
4. [Desde la VM bastión: cluster privado](#4-desde-la-vm-bastión-cluster-privado)
5. [TLS y DNS](#5-tls-y-dns)
6. [Verificación](#6-verificación)
7. [Backups (verificación)](#7-backups-verificación)
8. [Apagado (ahorro de costos)](#8-apagado-ahorro-de-costos)

---

## 1. Instalar las herramientas (en la máquina local)

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

## 2. Variables (completar una vez y pegar el bloque)

```bash
export PROJECT_ID="mi-proyecto-gcp"                 # project id de GCP
export REGION="us-central1"
export ZONE="us-central1-a"
export ADMIN_IP="$(curl -s -4 ifconfig.me)/32"      # IP pública IPv4 propia /32 (SSH al bastión). -4 fuerza IPv4: sin esa flag puede devolver IPv6, que admin_ip rechaza
export DOMAIN="academico.midominio.com"             # dominio real de la app
export ALERT_EMAIL="ops@midominio.com"
export BILLING_ACCOUNT_ID="XXXXXX-XXXXXX-XXXXXX"     # gcloud billing accounts list
export AWS_REGION="us-east-1"
```

> Estas variables son para la **máquina local** (§3). El bastión (§4) es otra shell y define las suyas por separado.

---

## 3. Desde la máquina local: infraestructura e imágenes

### 3.1 Autenticación

```bash
gcloud auth login
gcloud auth application-default login
gcloud config set project "$PROJECT_ID"
gcloud auth application-default set-quota-project "$PROJECT_ID"   # higiene de ADC para gcloud/otras libs; el budget de Terraform lo resuelve el provider (user_project_override en providers.tf)

# Bootstrap obligatorio: con user_project_override el provider atribuye la cuota
# a $PROJECT_ID, que debe tener la Cloud Resource Manager API habilitada. Terraform
# no puede habilitarla por sí mismo (la necesita para operar) → se habilita acá una vez.
gcloud services enable cloudresourcemanager.googleapis.com --project="$PROJECT_ID"

aws configure                                        # credenciales de bootstrap con permiso para crear S3 + IAM (NO las del usuario ops_backup, que Terraform crea en el apply)
```

### 3.2 Vincular facturación

```bash
gcloud billing projects link "$PROJECT_ID" --billing-account="$BILLING_ACCOUNT_ID"
```

### 3.3 Bucket de estado de Terraform (una sola vez)

```bash
gcloud storage buckets create "gs://tfstate-academico" \
  --project="$PROJECT_ID" --location="$REGION" --uniform-bucket-level-access
gcloud storage buckets update "gs://tfstate-academico" --versioning
```

> Si el nombre `tfstate-academico` ya está tomado (los buckets son globales), elegir otro y actualizar `bucket = "..."` en `infra/backend.tf`.

### 3.4 Variables de Terraform

```bash
cd infra
cat > terraform.tfvars <<EOF
project_id               = "$PROJECT_ID"
admin_ip                 = "$ADMIN_IP"
alert_email              = "$ALERT_EMAIL"
billing_account_id       = "$BILLING_ACCOUNT_ID"
app_host                 = "$DOMAIN"
aws_region               = "$AWS_REGION"
enable_app_metric_alerts = false   # se habilita al final, cuando la app ya recibe tráfico RPC (§6.1)
EOF
```

> `terraform.tfvars` está gitignored (no se commitea). `region`, `zone`, tamaños y budget usan defaults. `enable_app_metric_alerts` arranca en `false` porque las alertas dependen de métricas que la app emite recién al desplegarse.

### 3.5 Aplicar la infraestructura

```bash
terraform init
terraform plan      # revisar el plan
terraform apply     # escribir 'yes' para confirmar
```

### 3.6 Guardar los outputs

```bash
export AR_URL=$(terraform output -raw artifact_registry_url)
export CLUSTER=$(terraform output -raw cluster_name)
echo "Registry: $AR_URL"
echo "Cluster:  $CLUSTER"
cd ..
```

### 3.7 Construir y subir las imágenes al Artifact Registry

```bash
gcloud auth configure-docker "${REGION}-docker.pkg.dev"

docker build --provenance=false -t "$AR_URL/api:1.0.0" backend/
docker build --provenance=false -f frontend/Dockerfile -t "$AR_URL/web:1.0.0" .

docker push "$AR_URL/api:1.0.0"
docker push "$AR_URL/web:1.0.0"
```

> El overlay de prod NO se modifica acá. El `kustomization.yaml` trae las imágenes con el placeholder `PROJECT_ID`, que se reemplaza en el bastión al desplegar (§4.4) — así no hay que commitear ni pushear nada.

---

## 4. Desde la VM bastión: cluster privado

> **Importante (SSH):** dentro del bastión, pegar **un comando a la vez** y esperar a que vuelva el prompt antes del siguiente. El multi-paste por SSH se corrompe: mete saltos de línea en medio de los comandos y mezcla la salida con la entrada.

### 4.1 Entrar al bastión (en la máquina local)

```bash
gcloud compute ssh bastion --zone "$ZONE" --project "$PROJECT_ID"
```

### 4.2 Preparar el bastión (una sola vez, dentro de la sesión SSH)

El bastión es una shell nueva: no tiene las variables de §2. Definir las tres que necesita, **una por línea** (no pegar el bloque entero de §2). Reemplazar los valores por los propios:

```bash
PROJECT_ID="<tu-project-id>"
```

```bash
ZONE="us-central1-a"
```

```bash
ALERT_EMAIL="<tu-email>"
```

Verificar que quedaron limpias — los corchetes delatan saltos de línea o espacios de más:

```bash
printf '[%s]\n[%s]\n[%s]\n' "$PROJECT_ID" "$ZONE" "$ALERT_EMAIL"
```

Cada valor tiene que aparecer en su propia línea, sin nada extra. Si un `]` de cierre cae en otra línea, hay un salto de línea metido: volver a definir esa variable.

Instalar las herramientas:

```bash
sudo apt-get update
```

```bash
sudo apt-get install -y git kubectl google-cloud-cli-gke-gcloud-auth-plugin
```

Clonar el repo (URL literal, para no arrastrar saltos de línea en una variable):

```bash
git clone https://github.com/AlessandroRLM/infraestructura-y-servicios-cloud.git
```

```bash
cd infraestructura-y-servicios-cloud
```

Obtener las credenciales del cluster:

```bash
gcloud container clusters get-credentials gke-academico --zone "$ZONE" --project "$PROJECT_ID"
```

### 4.3 Crear el Secret de producción (`app-secrets`) (en el bastión)

Crear el namespace:

```bash
kubectl create namespace academico-prod --dry-run=client -o yaml | kubectl apply -f -
```

Crear el secret. La clave se ingresa por teclado con `read -s` (no se pega ni queda en el historial) y la misma variable va en la URL y en `POSTGRES_PASSWORD`, así no pueden quedar distintas. Es **una sola línea**: seleccionarla entera, pegarla y Enter — va a pedir la clave (usar solo letras y números: un `@`, `:` o `/` rompe la DSN de la URL).

```bash
read -s -p 'Clave: ' PGPASS && echo && kubectl -n academico-prod create secret generic app-secrets --from-literal=DATABASE_URL="postgres://app:${PGPASS}@postgres:5432/academico?sslmode=disable" --from-literal=POSTGRES_USER=app --from-literal=POSTGRES_PASSWORD="${PGPASS}" --from-literal=METRICS_AUTH_TOKEN="$(openssl rand -hex 24)" --dry-run=client -o yaml | kubectl apply -f -
```

> El comando es una sola línea física (sin `\` ni saltos) a propósito: el multi-paste por SSH inserta saltos en medio de las continuaciones y corrompe el bloque. En una línea, el pegado es seguro.

> Es idempotente (`--dry-run=client -o yaml | kubectl apply -f -`): si la clave quedó mal, se vuelve a correr el mismo comando y sobrescribe el secret. Pero `POSTGRES_PASSWORD` solo se aplica en la **primera** inicialización del volumen de postgres: si `postgres-0` ya arrancó con otra clave, además de recrear el secret hay que reinicializar el StatefulSet — `kubectl -n academico-prod scale statefulset postgres --replicas=0`, luego `kubectl -n academico-prod delete pvc data-postgres-0`, luego `scale --replicas=1` (borra el volumen: solo en bootstrap, sin datos reales).

> En producción endurecida esto se inyecta desde Secret Manager con External Secrets; acá se crea a mano para arrancar.

### 4.4 Apuntar el overlay al proyecto y desplegar (en el bastión)

El `kustomization.yaml` del repo trae las imágenes con el placeholder `PROJECT_ID` (ej. `us-central1-docker.pkg.dev/PROJECT_ID/academico/api`). Reemplazarlo por el project id real **en la copia clonada** — no se commitea ni pushea nada:

```bash
sed -i "s/PROJECT_ID/$PROJECT_ID/g" k8s/overlays/prod/kustomization.yaml
```

Verificar que quedó el project id real y en minúsculas (las referencias de imágenes Docker no aceptan mayúsculas):

```bash
grep newName k8s/overlays/prod/kustomization.yaml
```

> Si se usó una región distinta del default (`us-central1`), ajustar también el prefijo de región en esas líneas.

El overlay de prod trae el host del Ingress como placeholder `academico.example.com` en `patch-ingress-prod.yaml` (en las rules y en la sección TLS). Definir el dominio real y reemplazarlo — es el mismo dominio que irá en el registro DNS (§5.2):

```bash
DOMAIN="academico.midominio.com"   # el dominio real de la app
```

```bash
sed -i "s/academico.example.com/$DOMAIN/g" k8s/overlays/prod/patch-ingress-prod.yaml
```

Verificar que quedó el dominio real (debe aparecer en `host:` y en `hosts:`):

```bash
grep -E 'host' k8s/overlays/prod/patch-ingress-prod.yaml
```

Desplegar:

```bash
kubectl apply -k k8s/overlays/prod
```

Observar los pods hasta que `api`, `web`, `postgres-0` y `redis` estén `Running`. Reejecutar el comando cada tanto: las imágenes tardan en bajar:

```bash
kubectl -n academico-prod get pods
```

> Diagnóstico rápido por estado: `InvalidImageName` → el `sed` no reemplazó el placeholder (verificar que `$PROJECT_ID` no esté vacío y volver a aplicar). `ImagePullBackOff` → la imagen no está en el Artifact Registry (revisar §3.7). `CrashLoopBackOff` en `api` → suele ser que aún no conecta a Postgres; esperar a que `postgres-0` esté `Running`.

> La alerta de métricas de la app (`enable_app_metric_alerts`) se habilita al final, en §6.1, recién cuando la app ya recibe tráfico RPC.

### 4.5 Instalar el controlador ingress-nginx (en el bastión)

El `Ingress` del overlay usa `ingressClassName: nginx`, así que necesita el controlador ingress-nginx para obtener IP pública y enrutar el tráfico — y para que cert-manager resuelva el desafío http01 (§5). Instalar el manifiesto **cloud** (crea el namespace `ingress-nginx`, el controlador y un Service `LoadBalancer` que GCP expone con IP):

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.11.3/deploy/static/provider/cloud/deploy.yaml
```

> Usar siempre el manifiesto `cloud`, no `baremetal`: es el que crea el Service `LoadBalancer` en GKE. Si el tag da 404, tomar el de la última release en https://github.com/kubernetes/ingress-nginx/releases (el path es `.../controller-vX.Y.Z/deploy/static/provider/cloud/deploy.yaml`).

Esperar a que el Service del controlador reciba `EXTERNAL-IP`:

```bash
kubectl get svc -n ingress-nginx ingress-nginx-controller -w
```

Esa `EXTERNAL-IP` es la que va en el registro DNS (§5.2); el `Ingress academico` la hereda automáticamente.

---

## 5. TLS y DNS

### 5.1 cert-manager + certificado (Let's Encrypt) (en el bastión)

> Mismo bastión y misma sesión que §4. Si se reconectó, volver a definir `ALERT_EMAIL` (lo usa el ClusterIssuer de abajo) con `ALERT_EMAIL="<tu-email>"`.

Instalar cert-manager:

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml
```

Esperar a que el webhook esté listo:

```bash
kubectl -n cert-manager rollout status deploy/cert-manager-webhook
```

Crear el ClusterIssuer (bloque único — pegar entero):

```bash
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
```

Anotar el Ingress para que cert-manager emita el certificado `academico-prod-tls`:

```bash
kubectl -n academico-prod annotate ingress academico \
  cert-manager.io/cluster-issuer=letsencrypt --overwrite
```

### 5.2 DNS

En el bastión (es kubectl, y el cluster es privado), obtener la IP pública del Ingress — esperar 1-2 min a que se asigne:

```bash
kubectl -n academico-prod get ingress academico -o jsonpath='{.status.loadBalancer.ingress[0].ip}'; echo
```

Con esa IP, crear un registro **A** del dominio (`$DOMAIN`) apuntando a ella, en el proveedor de DNS / registrador. Cuando el DNS propague, cert-manager emite el certificado y la app queda en `https://$DOMAIN`.

---

## 6. Verificación

En el bastión, revisar los recursos de prod:

```bash
kubectl -n academico-prod get pods,svc,ingress
```

El certificado emitido por cert-manager:

```bash
kubectl -n academico-prod get certificate
```

Desde cualquier máquina, cuando DNS + cert estén listos, probar la app por HTTPS:

```bash
curl -s -o /dev/null -w "%{http_code}\n" "https://$DOMAIN/"
```

Dashboards: consola GCP → Monitoring → Dashboards (`infra`, `app`, `costos`), Budgets y Alerting (CPU, uptime; la de RPC errors se habilita en §6.1).

### 6.1 Habilitar la alerta de métricas de la app (último paso, en la máquina local)

La alerta de RPC error rate consulta `academico_rpc_requests_total`, una métrica que la app expone vía Managed Prometheus. Managed Prometheus rechaza alertas sobre métricas que nunca ingirió; por eso `enable_app_metric_alerts` arranca en `false`. Habilitarla recién cuando la app YA recibió tráfico RPC real — no basta con que el pod esté `Running`: el contador no tiene series hasta el primer request, y GMP tarda unos minutos en scrapear e ingerir.

**Prerrequisito (una vez):** el colector de Managed Prometheus necesita leer `app-secrets` para enviar el token en el scrape de `/metrics`. Aplicar el RBAC del colector — **aparte del kustomize** (el transformador de namespace reescribiría el subject `gmp-system` y rompería el binding):

```bash
kubectl -n academico-prod apply -f k8s/components/gmp/collector-rbac.yaml
kubectl -n gmp-system rollout restart daemonset/collector
```

> Sin esto, el colector falla con `secret watcher failed to start` y la métrica nunca se ingiere (el curl de abajo siempre devuelve `{}`).

Verificar primero que la métrica ya está en Cloud Monitoring. Debe devolver una línea con el `type`; si no devuelve nada, todavía no está (esperar unos minutos o generar tráfico):

```bash
TOKEN=$(gcloud auth print-access-token)
curl -s -G -H "Authorization: Bearer $TOKEN" -H "x-goog-user-project: $PROJECT_ID" "https://monitoring.googleapis.com/v3/projects/$PROJECT_ID/metricDescriptors" --data-urlencode 'filter=metric.type=starts_with("prometheus.googleapis.com/academico")'
```

Con la métrica ya ingerida, definir `enable_app_metric_alerts = true` en `terraform.tfvars`:

```hcl
enable_app_metric_alerts = true
```

Y reaplicar:

```bash
cd infra
terraform apply
```

> Si se aplica antes de que la métrica exista, Terraform falla con `PromQL metric(s) are invalid` — por eso la verificación previa.

---

## 7. Backups (verificación)

```bash
# ejecutar el backup manualmente desde la VM ops y verificar en ambas nubes
gcloud compute ssh ops --zone "$ZONE" --project "$PROJECT_ID" \
  --command 'sudo bash -c ". /etc/default/academico-backup && /opt/backup/backup.sh"'

gcloud storage ls "gs://$(cd infra && terraform output -raw gcs_backups_bucket)/"
aws s3 ls "s3://$(cd infra && terraform output -raw s3_backups_dr_bucket)/"
```

> La VM `ops` necesita las credenciales AWS configuradas (`aws configure`) — se proveen fuera de banda (no viven en el state). La prueba de restauración está en [`../infraestructura`](../infraestructura/README.md) §10 (`infra/scripts/restore.sh`).

---

## 8. Apagado (ahorro de costos)

```bash
cd infra
terraform destroy   # elimina toda la infra; el bucket de tfstate persiste
```

> El estado en GCS sobrevive, así que `terraform apply` reconstruye todo idéntico al retomar. Ver optimización de costos en [`../monitoreo-costos`](../monitoreo-costos/README.md).
