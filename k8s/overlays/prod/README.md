# Overlay: prod

Production overlay (namespace `academico-prod`) targeting GKE. Differs from `dev`/`test`:

- **Secret externo:** este overlay NO genera el `Secret`. `app-secrets` se provisiona fuera de banda (External Secrets Operator / GCP Secret Manager) con las claves `DATABASE_URL`, `METRICS_AUTH_TOKEN`, `POSTGRES_USER`, `POSTGRES_PASSWORD`. No se versionan credenciales de producción.
- **TLS:** el Ingress agrega `tls` con el secret `academico-prod-tls` (cert-manager / certificado gestionado). `COOKIE_SECURE` queda en `true` (default del base) — la cookie de sesión solo viaja por HTTPS.
- **Almacenamiento:** el PVC de postgres pasa a 20Gi en `standard-rwo` (pd-balanced).
- **Imágenes:** tags inmutables (`:1.0.0`), no `:dev`.
- **ResourceQuota:** límites de namespace dimensionados para el HPA de la API (hasta 6 réplicas).

## Placeholders a definir antes de aplicar

| Qué | Dónde | Valor actual (placeholder) |
|-----|-------|----------------------------|
| Dominio | `patch-ingress-prod.yaml` | `academico.example.com` |
| Secret TLS | `patch-ingress-prod.yaml` | `academico-prod-tls` (cert-manager) |
| Tag de imagen | `kustomization.yaml` | `1.0.0` |

```bash
kubectl kustomize k8s/overlays/prod   # render (no requiere cluster)
```
