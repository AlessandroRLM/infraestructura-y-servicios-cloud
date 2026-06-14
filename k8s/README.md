# Kubernetes — Academico

Manifiestos Kustomize para el stack `academico`. Estructura:

```
k8s/
├── base/               # Recursos base (agnósticos de entorno)
│   ├── api-deployment.yaml
│   ├── web-deployment.yaml
│   ├── postgres-statefulset.yaml
│   ├── redis-deployment.yaml
│   ├── services.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   ├── hpa.yaml
│   └── networkpolicy.yaml
└── overlays/
    └── dev/            # Overlay de desarrollo local (minikube)
        ├── kustomization.yaml      # secretGenerator + namespace + patches
        ├── namespace.yaml
        ├── patch-cookie-secure.yaml
        └── .env.dev                # credenciales dev — ignorado por git
```

> Overlays para `prod`/`test` y secrets gestionados externamente están fuera del scope de este slice.
> El fundamento de las decisiones de seguridad está en [`../docs/contenedores-kubernetes/seguridad-y-endurecimiento.md`](../docs/contenedores-kubernetes/seguridad-y-endurecimiento.md).

---

## Prerrequisitos

- [minikube](https://minikube.sigs.k8s.io/) ≥ 1.32
- [kubectl](https://kubernetes.io/docs/tasks/tools/) con soporte kustomize (`kubectl kustomize`)
- [Docker](https://docs.docker.com/get-docker/) para construir las imágenes

---

## 1. Iniciar minikube

```bash
minikube start --cni=calico
minikube addons enable ingress
minikube addons enable metrics-server   # requerido por HPA
```

> **NetworkPolicy enforcement**: las políticas requieren Calico (`--cni=calico`) en minikube o Dataplane V2 en GKE. El CNI por defecto (`kindnet`) no las aplica.

---

## 2. Construir imágenes

Construir directamente en el daemon Docker del nodo minikube evita el paso de carga y un problema conocido: `minikube image load` no reemplaza una imagen que ya existe con el mismo tag (deja la versión vieja en el nodo).

```bash
eval $(minikube -p minikube docker-env)

# API (Go, distroless nonroot)
docker build -t academico/api:dev backend/

# Web (React+Vite SPA, nginx unprivileged) — contexto = raíz del repo.
# Same-origin: no requiere build-arg de URL de API.
# --provenance=false produce una imagen de manifiesto único (la que espera el kubelet).
docker build --provenance=false -f frontend/Dockerfile -t academico/web:dev .
```

> Los deployments usan `imagePullPolicy: IfNotPresent`, así que el kubelet toma la imagen local del nodo.
> Para volver a una terminal apuntando al Docker del host: `eval $(minikube docker-env -u)`.

---

## 3. Desplegar

```bash
kubectl apply -k k8s/overlays/dev
```

Tras reconstruir una imagen (mismo tag), forzar el redeploy:

```bash
kubectl -n academico-dev rollout restart deploy/web deploy/api
```

---

## 4. Configurar `/etc/hosts`

```bash
echo "$(minikube ip) academico.local" | sudo tee -a /etc/hosts
```

---

## 5. Verificación

```bash
kubectl -n academico-dev get pods,svc,ingress

# Ruteo same-origin a través del Ingress:
curl http://academico.local/                         # → web (SPA), HTTP 200
curl -X POST -H 'Content-Type: application/json' -d '{}' \
  http://academico.local/auth.v1.AuthService/Login   # → api (error Connect), confirma el split

# /healthz a través del Ingress llega a web (nginx). El /readyz del api no se expone
# por el Ingress (solo los paths Connect /*.v1.* van al api); para verlo directo:
kubectl -n academico-dev port-forward deploy/api 8080:8080 &
curl http://localhost:8080/readyz                    # → 200 (requiere PG + Redis up)
```

---

## Secrets (dev)

El `Secret` de desarrollo se genera con `secretGenerator` a partir de `overlays/dev/.env.dev`, **ignorado por git** (`.gitignore` del overlay). El `kustomization.yaml` queda versionado; los valores no. Se usa `disableNameSuffixHash: true` para que el nombre estático `app-secrets` sea resolvible por los deployments.

Un `Secret` de Kubernetes guarda sus valores en base64, que no es cifrado. En producción se inyecta desde un proveedor externo (Vault, GCP Secret Manager, AWS Secrets Manager) vía ExternalSecret o CSI driver, con cifrado en reposo habilitado — fuera del scope de este slice.
