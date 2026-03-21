# TAN — Seguridad, Gobernanza y Riesgo

**Fecha:** 2026-02-22

---

## 1) Principios

- **Default seguro**: acciones riesgosas requieren aprobación
- **Least privilege**: scopes mínimos
- **Auditoría inmutable**: logs consultables y exportables
- **Defensa en profundidad**: auth + policy + budget + anomaly detection
- **Reversibilidad**: rollback y compensaciones

---

## 2) Matriz de riesgo por acción (ejemplo)

| Acción | Riesgo | Default | Mitigación |
|---|---:|---|---|
| Editar descripción | Bajo | ALLOW | validaciones |
| Subir imágenes | Bajo | ALLOW | antivirus/scan + size limits |
| Ajustar stock | Medio | ALLOW con límites | límites por SKU + rate limit |
| Publicar producto | Medio | APPROVAL | checklist + preview |
| Cambiar precio | Alto | APPROVAL o ALLOW limitado | bandas ±X%, alertas |
| Reembolso | Alto | APPROVAL | tope por monto + doble aprobación |
| Gastos en ads | Alto | APPROVAL | budget + whitelists |

---

## 3) Approvals (Human‑in‑the‑loop)

### 3.1 Tipos
- **Manual**: humano aprueba en UI
- **Policy‑auto**: reglas aprueban si cumple condiciones (ej: delta<2%)
- **Multi‑sig**: requiere 2 aprobaciones (refund alto)

### 3.2 Datos mínimos en una aprobación
- actor
- acción y payload
- impacto estimado (ej: delta de margen)
- recomendación del agente
- evidencia (inputs/links)

---

## 4) Presupuestos y rate limits

- Presupuesto por:
  - store
  - agente
  - conector (ads/pagos)
- Rate limits por endpoint y por acción
- Alertas por:
  - spikes (x3 baseline)
  - loops (misma acción repetida)
  - fallos repetidos de conector

---

## 5) Auditoría y pruebas forenses

- Logs encadenados por hash (opcional)
- Export a storage (WORM) para Enterprise
- Correlation IDs end‑to‑end

---

## 6) Cumplimiento y privacidad

- PII: encriptación en reposo, masking en logs
- Retención configurable (ej: 30/90/365 días)
- Acceso por RBAC (roles)
