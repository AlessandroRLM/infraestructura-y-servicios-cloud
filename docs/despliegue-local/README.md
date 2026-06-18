# Runbook — Pruebas y despliegue local

Guía end-to-end para levantar el sistema en local (minikube) y validar el IaC sin tocar la nube. Los detalles por capa están en [`../contenedores-kubernetes`](../contenedores-kubernetes/README.md), [`../../k8s/README.md`](../../k8s/README.md) e [`../infraestructura`](../infraestructura/README.md).

## Índice

1. [Qué corre en local y qué no](#1-qué-corre-en-local-y-qué-no)
2. [Prerrequisitos](#2-prerrequisitos)
3. [Kubernetes local (minikube)](#3-kubernetes-local-minikube)
4. [Validar el Terraform (sin apply)](#4-validar-el-terraform-sin-apply)
5. [Por qué `test`/`prod` y el monitoreo no corren en local](#5-por-qué-testprod-y-el-monitoreo-no-corren-en-local)
6. [Troubleshooting](#6-troubleshooting)

## 1. Qué corre en local y qué no

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

## 2. Prerrequisitos

| Herramienta | Versión usada |
| --- | --- |
| Docker | 29.x |
| minikube | 1.38.x |
| kubectl (kustomize integrado) | 1.36.x |
| Terraform | 1.15.x (≥ 1.6) |
| bun | 1.3.x (build del frontend) |
| go | 1.26.x (build del backend) |

Opcional, solo para `terraform plan` real: `gcloud` (`gcloud auth application-default login`) y `aws` (`aws configure`).

## 3. Kubernetes local (minikube)

### 3.1 Cluster + addons

```bash
minikube start --cni=calico            # Calico: enforcement real de NetworkPolicy
minikube addons enable ingress         # Ingress nginx
minikube addons enable metrics-server  # requerido por el HPA
```

### 3.2 Construir las imágenes en el daemon de minikube

Construir dentro del daemon del nodo evita `minikube image load`, que **no reemplaza una imagen con el mismo tag** (deja la vieja).

```bash
eval $(minikube -p minikube docker-env)

docker build -t academico/api:dev backend/

docker build --provenance=false -f frontend/Dockerfile -t academico/web:dev .
```

> `--provenance=false` produce una imagen de manifiesto único (la que espera el kubelet). El frontend es same-origin: no necesita build-arg de URL. Para volver al Docker del host: `eval $(minikube docker-env -u)`.

### 3.3 Desplegar el overlay dev

```bash
kubectl apply -k k8s/overlays/dev
kubectl -n academico-dev rollout status deploy/api
kubectl -n academico-dev rollout status deploy/web
```

### 3.4 Resolver el host del Ingress

```bash
echo "$(minikube ip) academico.local" | sudo tee -a /etc/hosts
```

### 3.5 Verificar

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

### 3.6 Redeploy tras reconstruir una imagen (mismo tag)

```bash
# reconstruir (paso 2.2) y luego:
kubectl -n academico-dev rollout restart deploy/web deploy/api
```

### 3.7 Datos de demo (seed)

El dataset de demo está en `infra/scripts/seed_demo.sql` (SQL escrito a mano, idempotente). El script `infra/scripts/seed.sh` lo envía a `psql` dentro del pod de postgres mediante `kubectl exec`, exactamente igual que el backup.

**Ejecutar desde la laptop contra minikube:**

```bash
NAMESPACE=academico-dev SQL_FILE=infra/scripts/seed_demo.sql infra/scripts/seed.sh
```

O bien directamente:

```bash
kubectl -n academico-dev exec -i statefulset/postgres -- \
  sh -c 'PGPASSWORD="$POSTGRES_PASSWORD" psql -v ON_ERROR_STOP=1 -h 127.0.0.1 -U "$POSTGRES_USER" -d "$POSTGRES_DB"' \
  < infra/scripts/seed_demo.sql
```

El SQL es idempotente: ejecutarlo más de una vez no duplica datos ni produce errores.

> **Requisito de base limpia.** `academic_periods` tiene `UNIQUE(year, term)`. El seed inserta los períodos 2025-1 y 2026-1 con UUID determinístico; si la base ya tiene un período con ese año/término creado por otra vía (la app o un seeder previo), el `INSERT` viola la unique y, con `ON_ERROR_STOP=1`, aborta toda la transacción. Sembrar contra una base limpia (minikube recién desplegado, o `kubectl -n academico-dev delete pvc` del volumen de postgres y volver a aplicar el overlay). Para re-sembrar una base ya sembrada por este script, ejecutar antes `seed_cleanup.sql`.

**Credenciales de acceso:**

| Email               | Contraseña   | Rol     | Acceso                                        |
| ------------------- | ------------ | ------- | --------------------------------------------- |
| admin@dev.local     | Admin1234!   | admin   | Todas las funciones                           |
| teacher@dev.local   | Teacher1234! | teacher | Calificaciones, reportes                      |
| student1@dev.local  | Student1234! | student | Calificaciones propias, inscripciones propias |
| student2@dev.local  | Student1234! | student | Calificaciones propias, inscripciones propias |
| student3@dev.local  | Student1234! | student | Calificaciones propias, inscripciones propias |

**Limpiar datos de demo** (elimina solo las filas sembradas; el admin no se elimina):

```bash
NAMESPACE=academico-dev SQL_FILE=infra/scripts/seed_cleanup.sql infra/scripts/seed.sh
```

### 3.8 Limpieza

```bash
kubectl delete -k k8s/overlays/dev    # borra el namespace y todo lo del overlay
minikube stop                         # apaga el cluster (conserva estado)
minikube delete                       # destruye el cluster
```

## 4. Validar el Terraform (sin apply)

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
terraform plan -var 'project_id=...' -var 'admin_ip=<ip-propia>/32'
```

> Sin credenciales, `plan` falla solo por auth (la config es válida): `could not find default credentials`. No se ejecuta `apply` en este entorno.

## 5. Por qué `test`/`prod` y el monitoreo no corren en local

- **GMP `PodMonitoring`** usa un CRD (`monitoring.googleapis.com/v1`) que solo existe en GKE; minikube lo rechazaría (`no matches for kind PodMonitoring`). Por eso vive en un component que solo incluyen `test`/`prod`.
- **`standard-rwo`** (pd-balanced) es una storage class de GKE; minikube usa `standard`.
- **TLS** del Ingress depende de certificados gestionados / cert-manager en GKE.
- **Secret de prod** se inyecta desde un gestor externo, no desde un `.env` local.

## 6. Troubleshooting

| Síntoma | Causa / arreglo |
| --- | --- |
| El pod corre código viejo tras rebuild | `minikube image load` no pisa el mismo tag → construir en `docker-env` (paso 2.2) |
| NetworkPolicy no bloquea nada | el CNI por defecto (kindnet) no las aplica → `minikube start --cni=calico` |
| HPA sin métricas | falta `minikube addons enable metrics-server` |
| `apply -k overlays/dev` falla con `PodMonitoring` | se está usando un overlay GKE; en local usar `dev` |
