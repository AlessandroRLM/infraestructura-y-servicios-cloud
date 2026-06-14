# Documentación del proyecto

Sistema de gestión académica (matrículas, notas, reportes) sobre arquitectura multi-cloud.
**GCP** como nube principal, **AWS** como respaldo/DR.

## Entregables

| Documento | Qué contiene |
|-----------|--------------|
| [Arquitectura](arquitectura/README.md) | Caso, requisitos, diagrama lógico multi-cloud, tabla de servicios, justificación de modelos. |
| [Infraestructura (runbook)](infraestructura/README.md) | Guía de despliegue paso a paso: VPC, subredes, firewall, VMs, storage, backups. |
| [Contenedores y Kubernetes](contenedores-kubernetes/README.md) | Empaquetado, manifiestos K8s, comunicación entre servicios, escalado y HA. |
| [Monitoreo y costos](monitoreo-costos/README.md) | Dashboards, alertas, estimación mensual y optimización de costos. |

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
