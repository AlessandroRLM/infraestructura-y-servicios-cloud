# Documentación del proyecto

Sistema de gestión académica (matrículas, notas, reportes) sobre arquitectura multi-cloud.
**GCP** como nube principal, **AWS** como respaldo/DR.

## Entregables

| Documento | Qué contiene |
|-----------|--------------|
| [Arquitectura](arquitectura/README.md) | Caso, requisitos, diagrama lógico multi-cloud, tabla de servicios, justificación de modelos. |
| [Guía de despliegue de infraestructura](infraestructura/README.md) | Diseño y evidencia: VPC, subredes, firewall, VMs, storage, backups. El procedimiento copy-paste está en *Despliegue en GCP*. |
| [Contenedores y Kubernetes](contenedores-kubernetes/README.md) | Empaquetado, manifiestos K8s, comunicación entre servicios, escalado y HA. |
| [Monitoreo y costos](monitoreo-costos/README.md) | Dashboards, alertas, estimación mensual y optimización de costos. |
| [Despliegue local](despliegue-local/README.md) | Pruebas y despliegue en local (minikube) + validación del IaC sin tocar la nube. |
| [Despliegue en la nube](despliegue-cloud/README.md) | Runbook copy-paste end-to-end del despliegue real (GCP + AWS): terraform, imágenes, app en GKE, TLS, backups. |

## Stack

| Capa | Tecnología |
|------|-----------|
| Frontend | React SPA (servido por Nginx, TLS en el Ingress) |
| Backend | Go (API Connect sobre HTTP) |
| Datos | PostgreSQL (StatefulSet), Redis (cache) |
| Orquestación | GKE (Kubernetes gestionado) |
| IaC | Terraform (GCP + AWS) |
| Observabilidad | Cloud Monitoring + Cloud Logging |
| Respaldo | AWS S3 (backups cross-cloud) |

## Convenciones

- Documentos concisos, enfocados en lo que la rúbrica evalúa.
- Diagramas en Mermaid (renderizan en GitHub).
- Las decisiones técnicas y tareas por módulo viven en el flujo SDD de cada capa, no acá.
- READMEs por módulo (`backend/`, `frontend/`, `infra/`): qué es y cómo se usa.

## Generar PDFs

El script `scripts/docs-pdf.sh` exporta todos los documentos a PDF con [pandoc](https://pandoc.org/) y preserva los diagramas Mermaid mediante [mermaid-filter](https://github.com/raghur/mermaid-filter).

```bash
bash scripts/docs-pdf.sh
```

Los PDFs se generan junto a cada `.md` (mismo directorio, extensión `.pdf`). Ver el encabezado del script para los requisitos de instalación (`pandoc`, `weasyprint`, `npm i -g mermaid-filter`).
