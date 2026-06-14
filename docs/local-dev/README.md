# Runbook — Pruebas y despliegue local

Guía end-to-end para levantar el sistema en local (minikube) y validar el IaC sin tocar la nube. Los detalles por capa están en [`../contenedores-kubernetes`](../contenedores-kubernetes/README.md), [`../../k8s/README.md`](../../k8s/README.md) e [`../infraestructura`](../infraestructura/README.md).

## 0. Qué corre en local y qué no

| Componente | Local (minikube) | Solo GKE / nube |
| --- | --- | --- |
| App completa (web, api, postgres, redis) — overlay `dev` | ✅ | — |
| Ruteo same-origin (Ingress nginx) | ✅ | — |
| NetworkPolicy (aislamiento) — requiere Calico | ✅ | — |
| HPA (autoescalado de api) — requiere metrics-server | ✅ | — |
| Terraform `fmt` / `validate` / `plan` | ✅ | — |
| Terraform `apply` (VPC, GKE, VMs, buckets, KMS) | — | ✅ nube real |
| Overlays `test` / `prod` (TLS, `standard-rwo`, secret externo) | — | ✅ GKE |
| GMP `PodMonitoring` + dashboards/alertas Cloud Monitoring | — | ✅ GKE |

> El overlay `dev` está pensado para minikube (HTTP, `.env.dev`, storage `standard` 1Gi). `test`/`prod` apuntan a GKE y no aplican en minikube (CRD de GMP, clase de storage `standard-rwo`, certificados TLS, secret externo).

## 1. Prerrequisitos

| Herramienta | Versión usada |
| --- | --- |
| Docker | 29.x |
| minikube | 1.38.x |
| kubectl (kustomize integrado) | 1.36.x |
| Terraform | 1.15.x (≥ 1.6) |
| bun | 1.3.x (build del frontend) |
| go | 1.26.x (build del backend) |

Opcional, solo para `terraform plan` real: `gcloud` (`gcloud auth application-default login`) y `aws` (`aws configure`).

## 2. Kubernetes local (minikube)

### 2.1 Cluster + addons

```bash
minikube start --cni=calico            # Calico: enforcement real de NetworkPolicy
minikube addons enable ingress         # Ingress nginx
minikube addons enable metrics-server  # requerido por el HPA
```

### 2.2 Construir las imágenes en el daemon de minikube

Construir dentro del daemon del nodo evita `minikube image load`, que **no reemplaza una imagen con el mismo tag** (deja la vieja).

```bash
eval $(minikube -p minikube docker-env)

docker build -t academico/api:dev backend/

docker build --provenance=false -f frontend/Dockerfile -t academico/web:dev .
```

> `--provenance=false` produce una imagen de manifiesto único (la que espera el kubelet). El frontend es same-origin: no necesita build-arg de URL. Para volver al Docker del host: `eval $(minikube docker-env -u)`.

### 2.3 Desplegar el overlay dev

```bash
kubectl apply -k k8s/overlays/dev
kubectl -n academico-dev rollout status deploy/api
kubectl -n academico-dev rollout status deploy/web
```

### 2.4 Resolver el host del Ingress

```bash
echo "$(minikube ip) academico.local" | sudo tee -a /etc/hosts
```

### 2.5 Verificar

```bash
# Todos los pods Ready (2 api · postgres · redis · 2 web)
kubectl -n academico-dev get pods

# Ruteo same-origin: / → web (SPA); path Connect → api
curl -s -o /dev/null -w "%{http_code}\n" http://academico.local/            # 200 (web)
curl -s -X POST -H 'Content-Type: application/json' -d '{}' \
  http://academico.local/auth.v1.AuthService/Login                          # error Connect (api)

# Aislamiento: un pod de otro namespace NO alcanza al api (Calico) → timeout
kubectl run probe --rm -i --restart=Never --image=busybox:1.36 -n default \
  -- wget -T5 -qO- http://api.academico-dev.svc:8080/healthz

# ResourceQuota del namespace
kubectl -n academico-dev get resourcequota academico-quota
```

### 2.6 Redeploy tras reconstruir una imagen (mismo tag)

```bash
# reconstruir (paso 2.2) y luego:
kubectl -n academico-dev rollout restart deploy/web deploy/api
```

### 2.7 Limpieza

```bash
kubectl delete -k k8s/overlays/dev    # borra el namespace y todo lo del overlay
minikube stop                         # apaga el cluster (conserva estado)
minikube delete                       # destruye el cluster
```

## 3. Validar el Terraform (sin apply)

```bash
cd infra
terraform fmt -recursive -check
terraform init -backend=false   # descarga providers, sin backend GCS
terraform validate              # → Success
```

`terraform plan` requiere credenciales reales (GCP ADC + AWS) y el backend de estado:

```bash
gcloud auth application-default login
aws configure
# el bucket de estado se crea una sola vez fuera de banda (ver docs/infraestructura §2)
terraform init                  # conecta el backend GCS
terraform plan -var 'project_id=...' -var 'admin_ip=<tu-ip>/32'
```

> Sin credenciales, `plan` falla solo por auth (la config es válida): `could not find default credentials`. No se ejecuta `apply` en este entorno.

## 4. Por qué `test`/`prod` y el monitoreo no corren en local

- **GMP `PodMonitoring`** usa un CRD (`monitoring.googleapis.com/v1`) que solo existe en GKE; minikube lo rechazaría (`no matches for kind PodMonitoring`). Por eso vive en un component que solo incluyen `test`/`prod`.
- **`standard-rwo`** (pd-balanced) es una storage class de GKE; minikube usa `standard`.
- **TLS** del Ingress depende de certificados gestionados / cert-manager en GKE.
- **Secret de prod** se inyecta desde un gestor externo, no desde un `.env` local.

## 5. Troubleshooting

| Síntoma | Causa / arreglo |
| --- | --- |
| El pod corre código viejo tras rebuild | `minikube image load` no pisa el mismo tag → construir en `docker-env` (paso 2.2) |
| NetworkPolicy no bloquea nada | el CNI por defecto (kindnet) no las aplica → `minikube start --cni=calico` |
| HPA sin métricas | falta `minikube addons enable metrics-server` |
| `apply -k overlays/dev` falla con `PodMonitoring` | estás usando un overlay GKE; en local usá `dev` |
