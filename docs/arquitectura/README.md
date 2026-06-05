# Arquitectura — Sistema de gestión académica

## 1. Caso y contexto

Un instituto profesional necesita llevar a la nube su sistema de **matrículas, notas y reportes**. Las exigencias del negocio son alta seguridad, trazabilidad de accesos y cambios, y presupuesto limitado.

La solución se despliega en **dos nubes públicas**:

- **GCP (principal):** ejecuta toda la aplicación sobre Kubernetes gestionado (GKE).
- **AWS (respaldo/DR):** almacena los backups fuera de la nube principal para recuperación ante desastres.

## 2. Requisitos

### 2.1 Funcionales

| ID   | Requisito                                                             |
| ---- | --------------------------------------------------------------------- |
| RF-1 | Gestión de matrículas (alta, baja, consulta de inscripciones).        |
| RF-2 | Registro y consulta de notas por alumno y asignatura.                 |
| RF-3 | Generación de reportes académicos.                                    |
| RF-4 | Autenticación y autorización por rol: administrador, docente, alumno. |
| RF-5 | Auditoría de accesos y cambios de configuración.                      |

### 2.2 No funcionales

| ID    | Requisito                                                   | Cómo se cumple                                                              |
| ----- | ----------------------------------------------------------- | --------------------------------------------------------------------------- |
| RNF-1 | Separación de ambientes dev/test/prod en la nube principal. | Namespaces aislados en GKE + ResourceQuotas + NetworkPolicies.              |
| RNF-2 | Cifrado en tránsito.                                        | TLS en Ingress (certificado gestionado), tráfico interno por red privada.   |
| RNF-3 | Cifrado en reposo.                                          | Discos GCE y buckets GCS/S3 cifrados (claves gestionadas).                  |
| RNF-4 | Logging central para auditoría.                             | Cloud Logging recibe logs de acceso y de actividad administrativa.          |
| RNF-5 | Backups diarios + prueba de restauración documentada.       | `pg_dump` diario → GCS → réplica a S3; restore probado y documentado.       |
| RNF-6 | Acceso administrativo controlado.                           | Único punto de entrada SSH/kubectl vía bastión; nodos en subredes privadas. |

### 2.3 SLA objetivo

| Métrica                             | Objetivo (producción) |
| ----------------------------------- | --------------------- |
| Disponibilidad                      | 99.5 % mensual        |
| RPO (pérdida máxima de datos)       | 24 h (backup diario)  |
| RTO (tiempo máximo de recuperación) | 4 h                   |

## 3. Diagrama lógico (multi-cloud)

```mermaid
flowchart TB
    user([Usuarios: admin / docente / alumno])

    subgraph GCP["GCP — nube principal (us-central1)"]
        lb["Ingress HTTPS<br/>(certificado gestionado)"]

        subgraph vpc["VPC vpc-academico"]
            subgraph pub["Subred pública"]
                bastion["VM bastion<br/>SSH / kubectl"]
            end
            subgraph priv["Subredes privadas"]
                subgraph gke["GKE — namespaces dev / test / prod"]
                    web["React (Nginx)"]
                    api["API Go"]
                    pg[("PostgreSQL<br/>StatefulSet + PVC")]
                    redis[("Redis<br/>cache")]
                end
                ops["VM ops<br/>cron de backup"]
            end
        end

        gcs[("GCS<br/>activos + backups")]
        mon["Cloud Monitoring<br/>+ Cloud Logging"]
    end

    subgraph AWS["AWS — respaldo / DR"]
        s3[("S3<br/>backups cross-cloud<br/>versionado + lifecycle")]
    end

    user --> lb --> web <--> api
    api <--> pg
    api <--> redis
    ops -->|pg_dump diario| gcs
    gcs -->|réplica| s3
    gke -.métricas/logs.-> mon
    bastion -.admin.-> gke
```

## 4. Diagrama integral de infraestructura cloud

Vista completa de punta a punta: ambas nubes, red, cómputo, almacenamiento, observabilidad, cifrado y el flujo de respaldo.

```mermaid
flowchart TB
    users([Usuarios])
    admin([Administrador])

    subgraph GCP["GCP — nube principal (us-central1)"]
        lb["Cloud Load Balancing<br/>Ingress HTTPS"]
        kms["Cloud KMS<br/>claves de cifrado"]
        mon["Cloud Monitoring<br/>+ Cloud Logging"]
        gcs[("GCS<br/>assets · backups · tfstate")]
        pd[("Persistent Disk<br/>PVC de postgres")]

        subgraph vpc["VPC vpc-academico"]
            subgraph subpub["subnet-public 10.0.0.0/24"]
                bastion["VM bastion<br/>e2-micro"]
            end
            subgraph subops["subnet-ops 10.0.1.0/24"]
                ops["VM ops<br/>e2-small · cron backup"]
            end
            subgraph subgke["subnet-gke 10.0.16.0/20"]
                subgraph cluster["GKE zonal · node pool e2-medium x2"]
                    subgraph nsprod["namespace prod"]
                        web["web · React/Nginx"]
                        api["api · Go/Connect"]
                        pg[("postgres")]
                        rd[("redis")]
                    end
                    nsdev["namespace dev"]
                    nstest["namespace test"]
                end
            end
            nat["Cloud NAT"]
        end
    end

    subgraph AWS["AWS — nube de respaldo"]
        iam["IAM · acceso mínimo"]
        s3[("S3 · backups DR<br/>versionado + lifecycle")]
    end

    users -->|HTTPS 443| lb --> web --> api
    api --> pg
    api --> rd
    pg -.volumen.-> pd
    admin -->|SSH IP fija| bastion
    bastion -.kubectl.-> cluster
    web -.assets.-> gcs
    ops -->|pg_dump diario| gcs
    gcs -->|réplica cross-cloud| s3
    iam -.autoriza.-> ops
    cluster -.métricas/logs.-> mon
    pg -.cifrado.-> kms
    gcs -.cifrado.-> kms
    bastion --> nat
    ops --> nat
    api --> nat
```

## 5. Servicios por nube

### GCP (principal)

| Servicio                   | Uso                                                                  |
| -------------------------- | -------------------------------------------------------------------- |
| GKE                        | Cluster Kubernetes que orquesta los contenedores.                    |
| Compute Engine (GCE)       | VM `bastion` (acceso) y VM `ops` (backups). Nodos del cluster.       |
| VPC + subredes             | Red privada segmentada (pública para bastión, privadas para cargas). |
| Cloud NAT                  | Salida a internet de las subredes privadas.                          |
| Persistent Disk            | Volúmenes persistentes (PVC) de PostgreSQL.                          |
| Cloud Storage (GCS)        | Activos estáticos, backups y estado de Terraform.                    |
| Cloud Load Balancing       | Expone la aplicación vía Ingress HTTPS.                              |
| Cloud Monitoring + Logging | Métricas, dashboards, alertas y logs de auditoría.                   |
| Cloud KMS                  | Claves de cifrado en reposo (CMEK donde aplique).                    |

### AWS (respaldo / DR)

| Servicio | Uso                                                                                |
| -------- | ---------------------------------------------------------------------------------- |
| S3       | Destino de los backups cross-cloud (versionado + lifecycle a almacenamiento frío). |
| IAM      | Credencial de acceso mínimo para que la VM `ops` escriba en el bucket.             |

## 6. Modelos de servicio

| Modelo   | Componentes                                                      | Por qué                                                                                                           |
| -------- | ---------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| **IaaS** | VMs `bastion` y `ops`, VPC, subredes, firewall.                  | Necesitamos control total de la red y del punto de acceso administrativo.                                         |
| **PaaS** | GKE (plano de control gestionado), Cloud Monitoring/Logging, S3. | Google/AWS operan el plano de control; nos enfocamos en la app, no en parchear masters ni servidores de métricas. |
| **SaaS** | —                                                                | No se usan SaaS de negocio; el sistema académico es propio.                                                       |

**Decisión clave:** GKE en lugar de un cluster autogestionado sobre VMs. El plano de control gestionado elimina el trabajo de operar y parchear los masters, reduce errores y libera tiempo para seguridad y observabilidad, que es donde el caso pone el foco. El costo del plano de control se evita usando un cluster zonal (gratis en el primer cluster por cuenta).

## 7. Modelo de despliegue

El despliegue es **nube pública multi-cloud**, no híbrido.

- **Híbrido** implica combinar nube pública con infraestructura privada/on-premise. No es el caso: no hay datacenter propio.
- Lo nuestro es **multi-cloud**: dos nubes públicas (GCP principal + AWS para DR).

**Por qué multi-cloud y no una sola nube:** el caso exige backups fuera de la nube principal para recuperación ante desastres. Guardar el respaldo en otra nube protege contra una falla total de cuenta o región del proveedor principal, que un backup en la misma nube no cubre. AWS S3 aporta almacenamiento de objetos barato, durable y con versionado, suficiente para el rol de respaldo sin sumar complejidad operativa.

## 8. Red y seguridad

### 8.1 Topología de red

```mermaid
flowchart LR
    inet([Internet])

    subgraph vpc["VPC vpc-academico (us-central1)"]
        subgraph s_pub["subnet-public 10.0.0.0/24"]
            bastion["VM bastion"]
        end
        subgraph s_gke["subnet-gke 10.0.16.0/20<br/>(pods/services en rangos secundarios)"]
            nodes["Nodos GKE"]
        end
        subgraph s_ops["subnet-ops 10.0.1.0/24"]
            ops["VM ops"]
        end
        nat["Cloud NAT"]
    end

    inet -->|SSH solo IP admin| bastion
    inet -->|HTTPS| nodes
    bastion --> nodes
    nodes --> nat --> inet
    ops --> nat
```

### 8.2 Plan de direccionamiento (CIDR)

| Rango                     | CIDR           | Uso                       |
| ------------------------- | -------------- | ------------------------- |
| subnet-public             | `10.0.0.0/24`  | VM bastión                |
| subnet-ops                | `10.0.1.0/24`  | VM ops                    |
| subnet-gke (nodos)        | `10.0.16.0/20` | Nodos del cluster         |
| Rango secundario pods     | `10.4.0.0/14`  | IPs de pods               |
| Rango secundario services | `10.8.0.0/20`  | ClusterIP de los Services |

Los rangos de pods y services se definen como rangos secundarios de la subred de GKE (IP aliasing), práctica nativa de GKE para asignar IPs enrutables a pods sin NAT interno.

### 8.3 Matriz de flujos de red

| Origen           | Destino            | Puerto    | Propósito                  |
| ---------------- | ------------------ | --------- | -------------------------- |
| Internet (admin) | bastión            | 22        | SSH administrativo         |
| Internet         | Load Balancer      | 443       | Acceso HTTPS a la app      |
| Load Balancer    | pods web/api       | 80 / 8080 | Tráfico de aplicación      |
| api              | postgres           | 5432      | Acceso a datos             |
| api              | redis              | 6379      | Sesiones y cache           |
| bastión          | API server de GKE  | 443       | `kubectl`                  |
| nodos / ops      | Internet (vía NAT) | 443       | Egress: imágenes, APIs, S3 |

### 8.4 Controles de seguridad

| Control                     | Implementación                                                                                |
| --------------------------- | --------------------------------------------------------------------------------------------- |
| Acceso administrativo       | SSH solo al bastión, restringido a la IP del administrador. Nodos y VM `ops` sin IP pública.  |
| Reglas de firewall          | Mínimo necesario: SSH al bastión, HTTPS al Ingress, tráfico interno explícito entre subredes. |
| Aislamiento entre ambientes | NetworkPolicies por namespace: dev/test no pueden alcanzar la base de prod.                   |
| Cifrado en tránsito         | TLS en el Ingress; tráfico entre pods dentro de la red privada del cluster.                   |
| Cifrado en reposo           | Discos persistentes y buckets cifrados; claves en Cloud KMS donde se requiera CMEK.           |
| Secretos                    | Kubernetes Secrets por namespace; credenciales de AWS solo en la VM `ops`.                    |
| Auditoría                   | Cloud Logging centraliza logs de acceso y de actividad administrativa (RNF-4).                |

### 8.5 Roles, permisos y pertenencia

La autorización es de dos capas: **permiso** (qué operación, data-driven) y **pertenencia a nivel de recurso** (sobre qué datos). Los permisos se guardan en tablas (`permissions`, `role_permissions`) y se asignan vía `user_roles`. El control real es del backend; la UI solo oculta lo que el rol no puede usar.

| Operación | Administrador | Docente | Alumno |
| --- | :---: | :---: | :---: |
| Gestionar usuarios, roles y permisos | Sí | — | — |
| Gestionar catálogo (programas, asignaturas, secciones) | Sí | — | — |
| Gestionar matrícula (anual) | Sí | — | — |
| Inscribir / anular secciones | Sí | — | — |
| Ver matrícula e inscripciones propias | Sí | — | Sí |
| Registrar / editar notas | Sí | Solo sus secciones | — |
| Ver notas de una sección | Sí | Solo sus secciones | — |
| Ver notas propias | Sí | — | Sí |
| Generar reportes | Sí | Solo sus secciones | — |
| Ver logs de auditoría | Sí | — | — |

**Dos capas de autorización:**

- El **permiso** habilita la operación (el interceptor lo verifica contra los permisos del rol); la **pertenencia** decide sobre qué recursos. Un docente con permiso para cargar notas solo puede hacerlo en las secciones que dicta (`section_teachers`).
- **Sin auto-acción:** nadie opera sobre sus propios registros académicos. Un docente que también es alumno no puede cargar ni editar la nota de una inscripción donde el alumno sea él mismo.
- **Admin total, pero auditado:** el administrador puede todo —incluida la corrección de una nota—, pero cada acción queda registrada en `audit_logs`. El poder no exime del rastro.

Regla de autorización para cargar o editar una nota (sobre la inscripción `SE`):

```text
permitir si:
  el rol incluye el permiso grades.write
    AND existe section_teachers(section = SE.section_id, teacher = user)  -- dicta la sección
    AND SE.student != user                                               -- no te calificás a vos mismo
  OR el rol incluye el permiso grades.override                           -- admin; queda auditado
```

## 9. Ambientes

Tres ambientes en un **único cluster GKE**, separados por namespace:

```mermaid
flowchart TB
    subgraph cluster["Cluster GKE (zonal)"]
        subgraph dev["namespace: dev"]
            d["web · api · postgres · redis"]
        end
        subgraph test["namespace: test"]
            t["web · api · postgres · redis"]
        end
        subgraph prod["namespace: prod"]
            p["web · api · postgres · redis"]
        end
    end
    rq["ResourceQuota por namespace"] -.-> dev & test & prod
    np["NetworkPolicy por namespace"] -.-> dev & test & prod
```

Un solo cluster mantiene el costo bajo (un único plano de control). El aislamiento es lógico, reforzado con ResourceQuotas (que dev no consuma recursos de prod) y NetworkPolicies (que los ambientes no se alcancen entre sí).

## 10. Modelo de datos

Entidades del dominio académico. Los nombres de tablas y columnas siguen la convención del código (inglés). La identidad y la autenticación viven en `users`; los datos de cada rol, en perfiles; y el control de acceso es data-driven (roles y permisos en tablas).

```mermaid
erDiagram
    users ||--o{ user_roles : ""
    roles ||--o{ user_roles : ""
    roles ||--o{ role_permissions : ""
    permissions ||--o{ role_permissions : ""
    users ||--o| student_profiles : ""
    users ||--o| teacher_profiles : ""
    teacher_profiles ||--o{ teacher_qualifications : ""
    programs ||--o{ student_profiles : ""
    programs ||--o{ program_courses : ""
    courses ||--o{ program_courses : ""
    courses ||--o{ sections : ""
    academic_periods ||--o{ sections : ""
    sections ||--o{ section_teachers : ""
    teacher_profiles ||--o{ section_teachers : ""
    student_profiles ||--o{ enrollments : ""
    enrollments ||--o{ section_enrollments : ""
    sections ||--o{ section_enrollments : ""
    section_enrollments ||--o{ grades : ""
    teacher_profiles ||--o{ grades : "graded_by"
    users ||--o{ audit_logs : ""

    users {
        uuid id PK
        string email UK
        string password_hash
    }
    roles {
        uuid id PK
        string name UK
    }
    permissions {
        uuid id PK
        string code UK
        string description
    }
    role_permissions {
        uuid role_id FK
        uuid permission_id FK
    }
    user_roles {
        uuid user_id FK
        uuid role_id FK
    }
    student_profiles {
        uuid user_id PK
        uuid program_id FK
        int admission_year
    }
    teacher_profiles {
        uuid user_id PK
        string department
        string title
    }
    teacher_qualifications {
        uuid id PK
        uuid teacher_id FK
        string degree
        int year
    }
    programs {
        uuid id PK
        string code UK
        string name
    }
    courses {
        uuid id PK
        string code UK
        string name
        int credits
    }
    program_courses {
        uuid program_id FK
        uuid course_id FK
    }
    academic_periods {
        uuid id PK
        int year
        int term
        date start_date
        date end_date
    }
    sections {
        uuid id PK
        uuid course_id FK
        uuid academic_period_id FK
        int capacity
    }
    section_teachers {
        uuid section_id FK
        uuid teacher_id FK
    }
    enrollments {
        uuid id PK
        uuid student_id FK
        int year
        string status "pending | paid | cancelled"
        timestamp paid_at
    }
    section_enrollments {
        uuid id PK
        uuid enrollment_id FK
        uuid section_id FK
        string status "in_progress | passed | failed | withdrawn"
        timestamp registered_at
    }
    grades {
        uuid id PK
        uuid section_enrollment_id FK
        uuid graded_by FK
        numeric value
        timestamp evaluated_at
    }
    audit_logs {
        uuid id PK
        uuid user_id FK
        string action
        string entity
        uuid entity_id
        string ip_address
        json before
        json after
        timestamp at
    }
```

Notas del modelo:

- **Identidad, roles y permisos:** `users` es solo identidad/auth. El acceso es data-driven: `roles` y `permissions` se relacionan vía `role_permissions` (M:N) y se asignan a usuarios vía `user_roles` (M:N) — una persona puede ser docente **y** alumno a la vez, y los permisos de cada rol se cambian sin tocar código.
- **Jerarquía académica:** `programs` ↔ `courses` es **M:N** (`program_courses`): una asignatura como "Inglés I" puede compartirse entre varias carreras. Cada `course` se dicta en `sections` (una sección = asignatura + `academic_period` + uno o varios docentes vía `section_teachers`, co-docencia).
- **Matrícula vs inscripción:** `enrollments` es la matrícula anual (financiera); `section_enrollments` son las inscripciones a secciones, y solo existen si hay matrícula vigente. Las notas (`grades`) cuelgan de la inscripción a la sección.
- **Pertenencia:** `section_teachers` define qué docentes dictan cada sección. Es la base de la autorización por pertenencia (ver §8.5).
- `audit_logs` da soporte a la trazabilidad (RF-5). Los reportes (RF-3) se generan por consulta y se cachean en Redis; no requieren tabla propia.

### 10.1 Convención de metadata y auditoría

Para no repetir columnas en el diagrama, estas se aplican por convención:

| Campo | Qué | Dónde |
| --- | --- | --- |
| `created_at` / `updated_at` | Cuándo se creó / modificó | Entidades mutables (users, perfiles, programs, courses, academic_periods, sections, enrollments, section_enrollments, grades). |
| `created_by` / `updated_by` | Quién (FK a users) | Cambios humanos sensibles: users, perfiles, programs, courses, sections, grades, enrollments. |
| `deleted_at` | Soft-delete (`NULL` = vivo) | Registros que no se borran físicamente: users, programs, courses, academic_periods, sections, enrollments, section_enrollments, grades. |
| `version` | Optimistic locking | `grades` (edición concurrente, evita pisar cambios). |
| `status` | Estado de negocio multi-estado | `enrollments` y `section_enrollments` (ya en el diagrama). |

Las tablas **append-only** (`audit_logs`, `user_roles`, `role_permissions`, `program_courses`, `section_teachers`) solo llevan `created_at`: no se editan, se insertan o se borran.

## 11. Flujos clave

### 11.1 Autenticación

```mermaid
sequenceDiagram
    participant U as Usuario
    participant A as API (Connect)
    participant P as PostgreSQL
    participant R as Redis

    U->>A: login (credenciales)
    A->>P: buscar usuario
    P-->>A: hash + roles
    A->>A: verificar (bcrypt)
    A->>R: crear sesión
    A-->>U: cookie httpOnly (id de sesión)

    Note over U,A: peticiones posteriores
    U->>A: petición + cookie
    A->>R: validar sesión (interceptor)
    R-->>A: usuario + roles/permisos
    A-->>U: respuesta autorizada
```

### 11.2 Inscripción de sección (con matrícula vigente)

```mermaid
sequenceDiagram
    participant U as Alumno / Admin
    participant A as API (Connect)
    participant R as Redis
    participant P as PostgreSQL

    U->>A: inscribir sección + cookie
    A->>R: validar sesión y permiso
    A->>P: ¿matrícula vigente del alumno en el año?
    alt sin matrícula
        A-->>U: rechazado (requiere matrícula)
    else con matrícula
        A->>P: insertar section_enrollment
        A-->>U: inscripción creada
    end
```

### 11.3 Carga de nota (con verificación de pertenencia)

```mermaid
sequenceDiagram
    participant T as Docente
    participant A as API (Connect)
    participant R as Redis
    participant P as PostgreSQL

    T->>A: cargar nota + cookie
    A->>R: validar sesión y permiso (grades.write)
    A->>P: ¿el docente dicta la sección? (section_teachers)
    A->>P: ¿el alumno de la inscripción no es el docente?
    alt no autorizado
        A-->>T: rechazado
    else autorizado
        A->>P: insertar grade
        A-->>T: nota registrada
    end
```

### 11.4 Generación de reporte (con cache)

```mermaid
sequenceDiagram
    participant U as Usuario
    participant A as API (Connect)
    participant R as Redis
    participant P as PostgreSQL

    U->>A: solicitar reporte
    A->>R: ¿cache?
    alt hit
        R-->>A: reporte cacheado
    else miss
        A->>P: consultar datos
        P-->>A: resultado
        A->>R: guardar en cache (TTL)
    end
    A-->>U: reporte
```

## 12. Tolerancia a fallos y alta disponibilidad

| Escenario                  | Mecanismo                                     | Impacto                                                                                                                                       |
| -------------------------- | --------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| Cae un pod                 | Liveness probe + Deployment lo recrea         | Nulo: las réplicas siguen atendiendo.                                                                                                         |
| Cae un nodo                | Pods reprogramados; el autoscaler agrega nodo | Breve, automático.                                                                                                                            |
| Cae la zona                | El cluster es zonal (una zona) → interrupción | Mitigación: recrear en otra zona con Terraform; los backups quedan intactos. Subir a cluster regional daría tolerancia de zona a mayor costo. |
| Datos borrados o corruptos | Restore desde el backup diario                | RPO 24 h, RTO 4 h.                                                                                                                            |
| Falla total de GCP         | DR desde el backup en S3 (AWS)                | Reconstrucción con Terraform + restore.                                                                                                       |

La elección de cluster zonal es un trade-off consciente de costo: una sola zona es suficiente para el SLA de 99.5 % del caso y mantiene el gasto bajo; si se exigiera mayor disponibilidad, se pasa a regional sin cambiar el diseño.

## 13. Dimensionamiento y capacidad

| Componente        | Tamaño                          | Justificación                                                                              |
| ----------------- | ------------------------------- | ------------------------------------------------------------------------------------------ |
| Nodos GKE         | `e2-medium` (2 vCPU / 4 GB) × 2 | Corren los 4 workloads de prod más dev/test a baja carga; HPA y autoscaler absorben picos. |
| VM `bastion`      | `e2-micro`                      | Solo gateway SSH; carga mínima; entra en la capa gratuita.                                 |
| VM `ops`          | `e2-small`                      | Cron de backup puntual, sin carga sostenida.                                               |
| PVC de PostgreSQL | 20 GB `pd-balanced`             | Volumen de datos académicos de un instituto: moderado.                                     |
| Redis             | Sin volumen                     | Cache y sesiones efímeras; se reconstruyen.                                                |

**Carga esperada:** un instituto del orden de cientos a pocos miles de alumnos, con concurrencia baja-media y picos puntuales (períodos de matrícula y cierre de notas). El HPA de la API cubre esos picos sin sobre-aprovisionar el resto del tiempo.

## 14. Respaldo y recuperación (DR)

```mermaid
sequenceDiagram
    participant cron as Cron (VM ops)
    participant pg as PostgreSQL (prod)
    participant gcs as GCS (GCP)
    participant s3 as S3 (AWS)

    cron->>pg: pg_dump (diario)
    pg-->>cron: dump comprimido
    cron->>gcs: subir backup
    cron->>s3: replicar a S3 (cross-cloud)
    Note over s3: versionado + lifecycle a almacenamiento frío
    Note over cron,s3: Restore probado y documentado (RNF-5)
```

- **Frecuencia:** diaria → RPO 24 h.
- **Doble destino:** GCS (rápido, misma nube) + S3 (otra nube, protege ante falla total de GCP).
- **Prueba de restauración:** documentada en el runbook de infraestructura (restaurar el dump en un namespace de prueba y validar integridad).

## 15. Decisiones y alternativas

| Decisión             | Elección                 | Alternativa                      | Por qué                                                                                             |
| -------------------- | ------------------------ | -------------------------------- | --------------------------------------------------------------------------------------------------- |
| Nube principal       | GCP                      | AWS                              | GKE es el Kubernetes gestionado más simple de operar que EKS.                                       |
| Orquestación         | GKE (gestionado)         | Kubernetes autogestionado en VMs | Evita operar y parchear el plano de control; menos riesgo.                                          |
| Topología de nubes   | Multi-cloud              | Una sola nube                    | El backup en otra nube protege ante falla total del proveedor principal.                            |
| Arquitectura backend | Monolito modular         | Microservicios                   | A esta escala, microservicios suman complejidad sin beneficio; el monolito modular queda extraíble. |
| Sesiones             | Server-side en Redis     | JWT stateless                    | Más simple y con revocación inmediata; el plazo manda. Aislado para migrar a JWT si hace falta.     |
| Persistencia         | `sqlc` + `pgx`           | ORM (gorm)                       | SQL type-safe y explícito; sin magia ni acoplamiento del ORM.                                       |
| Observabilidad       | Cloud Monitoring         | Prometheus + Grafana             | Nativo, no consume recursos del cluster y trae alertas de costo.                                    |
| Ambientes            | Namespaces en un cluster | Clusters o proyectos separados   | Aislamiento suficiente al menor costo.                                                              |
| IaC                  | Terraform                | Scripts gcloud/aws               | Reproducible; permite destruir y recrear para ahorrar.                                              |
