# 📄 DocForge API — Motor de Generación de Documentos PDF, Tirillas POS (80mm) & QR en Stream (Go 1.22)

**DocForge API** es un microservicio **100% Backend** de ultra-baja latencia escrito en **Go (Golang 1.22+)** sin dependencias externas pesadas (`CGO_ENABLED=0`), diseñado para generar **Facturas A4**, **Tirillas Térmicas POS (`80mm` y `58mm`)**, **Colillas de Pago de Nómina**, **Fórmulas Médicas** y **Códigos QR Vectoriales** directamente en **flujos de memoria (*Memory Streams*)** sin escribir un solo archivo temporal en el disco duro.

![Go 1.22](https://img.shields.io/badge/Go-1.22_Static_Binary-00ADD8?logo=go&logoColor=white)
![Memory Footprint](https://img.shields.io/badge/RAM_Footprint-%3C_20_MB-10B981)
![Zero Disk I/O](https://img.shields.io/badge/Stream_Engine-Zero_Disk_I%2FO_(sync.Pool)-F59E0B)
![Architecture](https://img.shields.io/badge/Architecture-Pipeline_%26_Hexagonal-8A2BE2)
![Design Patterns](https://img.shields.io/badge/GoF_Patterns-Template_Method_%7C_Abstract_Factory_%7C_Worker_Pool-06B6D4)
![OpenAPI 3.1](https://img.shields.io/badge/Docs-Scalar_OpenAPI_3.1-EC4899)

---

## ⚡ El Reto de Ingeniería que Resuelve

Generar documentos PDF en arquitecturas tradicionales (levantando instancias de Chromium/Puppeteer o guardando archivos temporales en `/tmp`) agota rápidamente la memoria RAM y los IOPS del servidor cuando decenas de clientes descargan facturas o tirillas POS al mismo tiempo.

**DocForge API** resuelve este cuello de botella mediante:
1. **Renderizado Directo sobre `*bytes.Buffer` (`PDF 1.4 Native Stream Builder`):** Construye la tabla de referencias cruzadas (`xref`), fuentes vectoriales y flujos de contenido en memoria en **`< 0.3 ms`** por documento.
2. **Reciclaje de Memoria con `sync.Pool` y `Worker Pool` Acotado:** Un pool de *Goroutines* con canales (`chan renderJob`) reutiliza los buffers de memoria ya reservados para evitar presión sobre el *Garbage Collector* de Go.
3. **Sello Criptográfico de Integridad (`SHA-256` + `HMAC-SHA256`):** Cada PDF emitido incluye en las cabeceras HTTP (`X-Document-SHA256`, `X-Document-HMAC`, `X-Render-Latency-Ms`, `X-Pool-Worker-Id`) su huella criptográfica verificable en `GET /api/v1/documents/verify/{hash}`.

---

## 🧩 Patrones de Arquitectura y de Diseño (GoF)

- **`Template Method Pattern` (`internal/application/template/`):** Define el algoritmo invariante de generación documental (`Validate` $\rightarrow$ `CreateGeometry` $\rightarrow$ `PopulateBuilder` $\rightarrow$ `WithSecurityFooter` $\rightarrow$ `WriteToBuffer`), con implementaciones para `InvoiceA4Template`, `PosThermalTicketTemplate`, `PayrollStubTemplate` y `MedicalOrderTemplate`.
- **`Abstract Factory Pattern` (`internal/domain/factory/`):** Provee las familias geométricas exactas para `A4_PORTRAIT` (`595x842 pt`), `THERMAL_80MM` (`226 pt`) y `THERMAL_58MM` (`164 pt`).
- **`Fluent Builder Pattern` (`internal/infrastructure/pdf/`):** Construcción fluida de encabezados, tablas e ítems sobre el stream PDF.
- **`Worker Pool + Object Pool Pattern` (`internal/application/pool/`):** Concurrencia controlada con canales y `sync.Pool`.

---

## 🐳 Despliegue Rápido (Docker Compose & ALpanel)

```bash
cp .env.example .env
docker compose up -d --build
```

- **Documentación Interactiva Scalar OpenAPI 3.1:** `http://localhost:8086/` (o `/docs`)
- **Health & Métricas del Worker Pool:** `http://localhost:8086/api/v1/health`
