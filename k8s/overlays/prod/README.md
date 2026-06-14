# Overlay: prod

Production overlay (namespace `academico-prod`) targeting GKE. Differs from `dev`/`test`:

- **Secret externo:** este overlay NO genera el `Secret`. `app-secrets` se provisiona fuera de banda (External Secrets Operator / GCP Secret Manager) con las claves `DATABASE_URL`, `METRICS_AUTH_TOKEN`, `POSTGRES_USER`, `POSTGRES_PASSWORD`. No se versionan credenciales de producción.
- **TLS:** el Ingress agrega `tls` con el secret `academico-prod-tls` (cert-manager / certificado gestionado). `COOKIE_SECURE` queda en `true` (default del base) — la cookie de sesión solo viaja por HTTPS.
- **Almacenamiento:** el PVC de postgres pasa a 20Gi en `standard-rwo` (pd-balanced).
- **Imágenes:** tags inmutables (`:1.0.0`), no `:dev`.
- **ResourceQuota:** límites de namespace dimensionados para el HPA de la API (hasta 6 réplicas).

## Imágenes

Las imágenes en `kustomization.yaml` referencian el Artifact Registry. La URL base proviene del output de Terraform `artifact_registry_url`. Antes de desplegar a producción, reemplazar `PROJECT_ID` (y la región si no es `us-central1`) en `kustomization.yaml`, o usar `kustomize edit set image`:

```bash
cd k8s/overlays/prod
kustomize edit set image academico/api=$(terraform -chdir=../../../infra output -raw artifact_registry_url)/api:1.0.0
kustomize edit set image academico/web=$(terraform -chdir=../../../infra output -raw artifact_registry_url)/web:1.0.0
```

Flujo de build y push (requiere `gcloud auth configure-docker us-central1-docker.pkg.dev` previo):

```bash
docker tag academico/api:dev <artifact_registry_url>/api:1.0.0
docker push <artifact_registry_url>/api:1.0.0

docker tag academico/web:dev <artifact_registry_url>/web:1.0.0
docker push <artifact_registry_url>/web:1.0.0
```

## Placeholders a definir antes de aplicar

| Qué | Dónde | Valor actual (placeholder) |
|-----|-------|----------------------------|
| Dominio | `patch-ingress-prod.yaml` | `academico.example.com` |
| Secret TLS | `patch-ingress-prod.yaml` | `academico-prod-tls` (cert-manager) |
| Tag de imagen | `kustomization.yaml` | `1.0.0` |

```bash
kubectl kustomize k8s/overlays/prod   # render (no requiere cluster)
```
