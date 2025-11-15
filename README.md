# Traefik Visitor Tracking Middleware

Eine Traefik-Middleware zum Tracken von Website-Besuchern mit InfluxDB-Integration.

## Features

- **IP-Adresse**: Gehashed mit Salt für Datenschutz
- **Timestamp**: Datum und Uhrzeit des Besuchs
- **Domain**: Besuchte Domain
- **Pfad**: Aufgerufener URL-Pfad
- **User-Agent**: Browser-Information
- **InfluxDB Integration**: Direkte Speicherung in InfluxDB v2

## Installation

### 1. Plugin in Traefik einrichten

In Ihrer Traefik-Konfiguration (`traefik.yml`) das Plugin hinzufügen:

```yaml
experimental:
  plugins:
    visitor-tracking:
      moduleName: github.com/samuelerb/traefik-visitor-middleware
      version: v1.0.0
```

### 2. InfluxDB starten

```bash
cd /path/to/traefik-visitor-middleware
docker-compose up -d
```

### 3. Middleware in dynamischer Konfiguration

In `/etc/traefik/conf.d/dynamic_conf.yml`:

```yaml
http:
  middlewares:
    visitor-tracking:
      plugin:
        visitor-tracking:
          influxdb_url: "http://influxdb-visitor-tracking:8086"
          influxdb_token: "visitor-tracking-token"
          influxdb_org: "visitors-org"
          influxdb_bucket: "visitors"
          hash_salt: "your-random-salt-string"
```

### 4. Middleware zu Services hinzufügen

```yaml
http:
  routers:
    your-service:
      rule: "Host(`example.com`)"
      middlewares:
        - visitor-tracking
        - other-middleware
      service: your-service
```

## Konfiguration

| Parameter | Beschreibung | Default |
|-----------|--------------|---------|
| `influxdb_url` | InfluxDB Server URL | `http://localhost:8086` |
| `influxdb_token` | InfluxDB Auth Token | **erforderlich** |
| `influxdb_org` | InfluxDB Organisation | **erforderlich** |
| `influxdb_bucket` | InfluxDB Bucket | `visitors` |
| `hash_salt` | Salt für IP-Hashing | `default-salt` |

## InfluxDB Datenstruktur

Die Daten werden in InfluxDB mit folgender Struktur gespeichert:

- **Measurement**: `visitor_tracking`
- **Tags**: `domain`, `path`, `ip_hash`, `user_agent`
- **Fields**: `visit_count` (immer 1)
- **Timestamp**: Besuchszeit

## Beispiel-Query in InfluxDB

```flux
from(bucket: "visitors")
  |> range(start: -1d)
  |> filter(fn: (r) => r["_measurement"] == "visitor_tracking")
  |> group(columns: ["domain"])
  |> count()
```