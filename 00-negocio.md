# Tienda Agent‑Native (TAN) — Documento de Negocio (No Técnico)

**Fecha:** 2026-02-22  
**Autor:** Nicolas Roldan (con asistencia)  

## 1) Resumen ejecutivo

**Tienda Agent‑Native (TAN)** es una plataforma tipo Shopify, pero diseñada para que **agentes autónomos de IA** puedan **crear, operar y optimizar** un e‑commerce end‑to‑end (catálogo, stock, pedidos, atención, promos) de forma segura.

La diferencia clave con Shopify no es “la tienda”: es la **capa agent‑native** que permite que software autónomo ejecute operaciones reales sin generar caos: **permisos**, **límites**, **aprobaciones**, **auditoría**, **presupuestos**, y **reputación**.

En paralelo, TAN expone un **endpoint estándar para agentes** (catálogo y compra legible por máquina) y puede listar tiendas en un **registry** para discovery: de esta manera, **otros agentes** pueden encontrar la tienda y comprar.

---

## 2) El problema que resolvemos

A medida que los agentes de IA se vuelven capaces de operar herramientas, aparecen dos fricciones grandes:

### a) Operar un e‑commerce hoy requiere humanos o equipos
Un e‑commerce típico tiene tareas continuas:
- Carga y mantenimiento de catálogo (títulos, imágenes, atributos, variantes)
- Gestión de stock y reposición
- Gestión de pedidos, devoluciones, comunicaciones
- Pricing y promociones
- Marketing y campañas
- Atención al cliente y post‑venta

Esto es costoso, lento y difícil de escalar.

### b) La autonomía sin controles es riesgosa
Si un agente puede “tocar” precios, publicar productos o devolver dinero:
- aumenta el riesgo de errores caros
- aumenta fraude/abuso
- complica compliance y responsabilidad

---

## 3) Propuesta de valor (para negocio)

### Para comerciantes / marcas
- **Time‑to‑launch**: una tienda puede nacer rápido (setup guiado por agente)
- **Menos costo operativo**: automatiza tareas repetitivas 24/7
- **Más performance**: ciclos de optimización (catálogo, precios, promos) con reglas
- **Menos errores**: políticas, límites y auditoría por defecto

### Para agentes compradores (futuro)
- Compras automáticas B2B/B2C a través de un **protocolo de comercio agent‑friendly**
- Mejor discovery y confianza (registry + señales de reputación)

### Para el ecosistema (partners)
- Integraciones “plug‑and‑play” (envíos, pagos, ads, ERPs)
- A futuro: marketplace de conectores y módulos, pero **no** es el core inicial

---

## 4) Qué construimos (en simple)

TAN ofrece 3 cosas:

1) **Storefront humano**  
Una web normal (como Shopify) para que humanos vean la tienda y compren.

2) **Agent Commerce Endpoint (ACE)**  
Un “puerto” estándar para que agentes:
- lean catálogo (productos, stock, precios)
- coticen envío
- armen carrito
- creen orden
- paguen (o soliciten aprobación)

3) **Capa de control y confianza**  
- permisos por acción
- límites de precio/stock/devoluciones
- aprobaciones humanas/policy engine
- auditoría firmada de acciones
- presupuestos y alertas

---

## 5) Cómo se encuentra la tienda (discovery)

**Etapa 1 (MVP):** por URL (web + endpoint estándar)  
**Etapa 2:** registry/directorio de tiendas agent‑native con búsqueda y verificación  
**Etapa 3:** indexación abierta (feeds, ranking, ecosistema)

---

## 6) Modelo de negocio (opciones)

> Recomendación: arrancar con un modelo simple, y evolucionar.

### 6.1 Suscripción SaaS (tipo Shopify)
- Planes por tienda (Basic/Pro/Enterprise)
- Límites por volumen (productos, órdenes, usuarios/agentes, conectores)
- Add‑ons (automations avanzadas, auditoría premium, multi‑store)

**Pros:** predecible, fácil de explicar  
**Contras:** no captura bien el valor del “uso” de agentes

### 6.2 Take rate / comisión por transacción
- % sobre GMV o fee por orden

**Pros:** alineado a éxito  
**Contras:** sensibilidad a márgenes; regulación; comparaciones con marketplaces

### 6.3 Pricing agent‑native por uso (recomendado como add‑on)
- Cobro por tareas (ej: “publicación masiva”, “optimización de catálogo”)
- Cobro por ejecuciones (workflows)
- Presupuestos y controles por cuenta

**Pros:** encaja con agentes; permite margen por automatización  
**Contras:** requiere buena medición y observabilidad

### 6.4 Servicios / partners (go‑to‑market)
- Implementación y onboarding para merchants
- Integradores certificados

---

## 7) Estrategia de entrada (GTM)

Arrancar con verticales donde el riesgo operativo sea menor y el ROI sea obvio:

1) **Catálogo + stock** (alto volumen, bajo riesgo financiero directo)  
2) **Órdenes + comunicación** (beneficio claro, controlable con approvals)  
3) **Optimización** (pricing/promos/ads con límites estrictos)

---

## 8) Principios de diseño (para no romper confianza)

- **Safety by default**: acciones sensibles en modo “proponer” + aprobación
- **Least privilege**: permisos mínimos necesarios
- **Auditabilidad**: todo cambio importante queda registrado
- **Observabilidad**: costos, errores, loops, anomalías
- **Reversibilidad**: rollback/corrección simple

---

## 9) Qué NO es (para evitar confusión)

- No es un “marketplace para vender agentes” (eso puede existir después)
- No es sólo una API: también tiene storefront humano
- No reemplaza al merchant: lo potencia con un “operador autónomo” con reglas

---

## 10) Glosario rápido (no técnico)

- **Agente**: software que puede decidir y ejecutar pasos
- **Capability**: una habilidad concreta (ej: publicar productos)
- **Policy / permisos**: reglas que limitan qué puede hacer
- **Aprobación**: un paso humano o automático antes de ejecutar
- **ACE**: endpoint para que agentes consulten y compren
- **Registry**: directorio para que agentes encuentren tiendas
