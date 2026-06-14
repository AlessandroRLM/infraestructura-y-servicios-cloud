# Seguridad y endurecimiento (hardening) de los manifiestos

Fundamento de las decisiones de seguridad aplicadas a los manifiestos de `k8s/` y al empaquetado de contenedores. Cada control se justifica con su fuente oficial.

El objetivo es que el `base` cumpla el perfil **Restricted** de los Pod Security Standards de Kubernetes, que codifica las prácticas de endurecimiento vigentes para cargas sin requisitos de privilegio.

## 1. Contexto de seguridad de los pods (Pod Security Standards: Restricted)

Cada contenedor (incluido el `initContainer`) declara:

| Campo | Valor | Qué hace |
| --- | --- | --- |
| `runAsNonRoot` | `true` | El kernel rechaza el contenedor si su imagen intenta correr como UID 0. Reduce el blast radius de un escape de contenedor. |
| `allowPrivilegeEscalation` | `false` | Impide ganar privilegios vía binarios `setuid`/`setgid` (anula `no_new_privs`). |
| `readOnlyRootFilesystem` | `true` | El filesystem raíz es inmutable; un atacante no puede dejar binarios ni modificar la imagen en runtime. Lo que necesita escritura se monta como `emptyDir` (`/tmp`). |
| `capabilities.drop: ["ALL"]` | — | Quita todas las capabilities de Linux; ninguna de estas cargas necesita capabilities. |
| `seccompProfile.type: RuntimeDefault` | (a nivel pod) | Aplica el perfil seccomp del runtime, que bloquea syscalls peligrosas. |

El perfil **Restricted** exige exactamente este conjunto: `capabilities.drop` debe incluir `ALL`, `runAsNonRoot` debe ser `true`, `seccompProfile.type` debe ser `RuntimeDefault` o `Localhost`, y `allowPrivilegeEscalation` debe ser `false`.

`readOnlyRootFilesystem` no lo exige el perfil Restricted, pero sí lo recomienda la lista de verificación de seguridad de Kubernetes y la guía de endurecimiento de la NSA/CISA.

> El `api` (binario Go sobre `distroless:nonroot`) y el `initContainer` (`busybox`, UID 65534) corren con root de solo lectura sin montajes extra salvo `/tmp`. El `web` necesita además `/tmp` por nginx (ver §2).

Referencias:
- [Pod Security Standards — Kubernetes](https://kubernetes.io/docs/concepts/security/pod-security-standards/)
- [Configure a Security Context for a Pod or Container — Kubernetes](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)
- [Application Security Checklist — Kubernetes](https://kubernetes.io/docs/concepts/security/application-security-checklist/)

## 2. nginx sin privilegios (web)

La imagen oficial `nginx` arranca su proceso maestro como **root** para poder bindear el puerto 80. Eso contradice `runAsNonRoot`. Se reemplaza por la imagen oficial **`nginxinc/nginx-unprivileged`**, que:

- escucha en **8080** en vez de 80,
- corre como **UID 101** (sin root).

Por eso el `Deployment` de `web` declara `runAsUser: 101`, `containerPort: 8080` y tres `emptyDir` para los paths que nginx escribe en runtime: `/var/cache/nginx` (`client_temp` y demás directorios temporales), `/run` (`nginx.pid`) y `/tmp`. Sin estos montajes, con `readOnlyRootFilesystem: true` nginx falla al iniciar. El `Service` de `web` mantiene `port: 80` y apunta a `targetPort: 8080`, así el Ingress y el resto del cluster siguen hablando por 80.

Referencia:
- [nginx/docker-nginx-unprivileged — README](https://github.com/nginx/docker-nginx-unprivileged/blob/main/README.md)

## 3. NetworkPolicy: deny por defecto en ingress y egress

Un pod sin NetworkPolicy acepta todo el tráfico entrante y saliente. Las políticas definen un modelo de **denegar por defecto y permitir explícitamente**, en ambas direcciones:

- `default-deny-ingress` y `default-deny-egress`: con `podSelector: {}` aíslan todos los pods del namespace. Una política de egress con la lista vacía bloquea todo el tráfico saliente.
- `allow-same-namespace`: permite tráfico entre pods del propio namespace (ingress).
- `allow-ingress-nginx`: permite que el controlador Ingress (namespace `ingress-nginx`) alcance `web` y `api`.
- `allow-dns-egress`: habilita la resolución de nombres hacia `kube-dns` (UDP/TCP 53) en `kube-system`. **Sin esto, con egress denegado, ningún pod resuelve DNS** y todo se rompe.
- `allow-api-egress`: el `api` solo puede salir hacia `postgres:5432` y `redis:6379`; nada más.

El egress importa tanto como el ingress: sin él, un pod comprometido puede exfiltrar datos a internet o alcanzar el endpoint de metadatos del proveedor cloud. Restringir el egress al mínimo (DNS + dependencias declaradas) cierra esa vía.

**Enforcement**: las NetworkPolicy las aplica el CNI, no Kubernetes. El CNI por defecto de minikube (kindnet) no las aplica. Por eso este proyecto usa **Calico** en minikube (`minikube start --cni=calico`) y **Dataplane V2** en GKE. Ver [`docs/infraestructura`](../infraestructura/README.md) para el lado GKE.

Referencias:
- [Network Policies — Kubernetes](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [Declare Network Policy — Kubernetes](https://kubernetes.io/docs/tasks/administer-cluster/declare-network-policy/)

## 4. Secrets fuera del control de versiones

Los `Secret` de Kubernetes guardan sus valores en **base64, que no es cifrado**: cualquiera que lea el manifiesto recupera el valor en claro. Por eso no se versionan credenciales.

El overlay `dev` genera el `Secret` con `secretGenerator` a partir de `.env.dev`, archivo **ignorado por git** (`.gitignore` del overlay). El `kustomization.yaml` queda versionado; los valores, no. Kustomize codifica a base64 automáticamente, así que `.env.dev` se escribe en texto plano.

Para staging/producción no se usan literales ni archivos en el repo: el `Secret` se inyecta desde un gestor externo (Secret Manager, External Secrets, sealed-secrets) y se habilita cifrado en reposo (encryption at rest) en el cluster.

Referencias:
- [Good practices for Kubernetes Secrets — Kubernetes](https://kubernetes.io/docs/concepts/security/secrets-good-practices/)
- [Managing Secrets using Kustomize — Kubernetes](https://kubernetes.io/docs/tasks/configmap-secret/managing-secret-using-kustomize/)

## 5. `COOKIE_SECURE` seguro por defecto

El `ConfigMap` base define `COOKIE_SECURE: "true"` (la cookie de sesión solo viaja por HTTPS). El overlay `dev` lo baja a `"false"` con un patch, porque minikube corre sobre HTTP plano. El default seguro vive en el `base`; relajarlo es una decisión explícita y local de cada overlay, no un olvido.

## 6. `.dockerignore` (higiene del build context)

El build de la imagen `web` usa la **raíz del repo** como contexto (buf lee `../backend/proto`). Sin `.dockerignore`, Docker envía todo el repo al daemon: `.git`, `infra/`, `docs/`, `node_modules`, etc. Eso ralentiza el build, invalida el cache de capas ante cualquier cambio y arriesga filtrar archivos sensibles a una capa de imagen. El `.dockerignore` deja pasar solo `frontend/` y `backend/proto`.

Referencia:
- [Build context — Docker Docs](https://docs.docker.com/build/concepts/context/)

## 7. Disponibilidad: anti-afinidad de pods

`api` y `web` corren con 2 réplicas y `podAntiAffinity` (preferida, no obligatoria) por `kubernetes.io/hostname`: el scheduler intenta repartir las réplicas en nodos distintos para que la caída de un nodo no tumbe toda una capa. Es preferida para no bloquear el scheduling en un cluster de un solo nodo (minikube).

Referencia:
- [Assigning Pods to Nodes — Kubernetes](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)

## Diferido a overlays de producción

Fuera del alcance de este slice (dev); pendiente para staging/prod:

- `fsGroup` en `postgres` y `redis` para correr esos contenedores sin root (sus imágenes hoy arrancan como root para ajustar permisos del volumen).
- `PodDisruptionBudget` para `api` y `web`.
- Cifrado en reposo de Secrets (encryption at rest) e inyección desde un gestor externo.
- Secrets generados con sufijo de hash (rollout automático al rotar credenciales).
