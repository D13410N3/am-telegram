# AM-Telegram

A lightweight service that forwards Prometheus AlertManager alerts to Telegram chats.

## Overview

AM-Telegram receives webhook notifications from AlertManager, processes them according to configured rules, and forwards them to specified Telegram chats. The service provides:

- Alert filtering based on status and time of day
- Custom emoji indicators based on alert severity
- Flexible recipient configuration
- Prometheus metrics for monitoring
- Structured JSON logging

## Features

- **Smart Alert Routing**: Configure default recipients and override or add additional recipients per alert
- **Working Hours Filter**: Option to send alerts only during specified working hours (8:00-22:00)
- **Resolved Alert Control**: Option to suppress resolved alert notifications
- **Rich Formatting**: Includes links to Grafana dashboards, AlertManager silence UI, and original alert source
- **Prometheus Metrics**: Track alert volume, delivery success/failure rates
- **Health Check Endpoint**: For monitoring service health

## Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `TELEGRAM_BOT_TOKEN` | Telegram Bot API token | Yes | - |
| `DEFAULT_RECEPIENTS` | Default Telegram chat IDs (comma-separated) | Yes | - |
| `GRAFANA_BASE_URL` | Base URL for Grafana links | Yes | - |
| `PROM_BASE_URL` | Base URL for Prometheus links | Yes | - |
| `AM_BASE_URL` | Base URL for AlertManager links | Yes | - |
| `LISTEN_ADDR` | Server listen address | No | `:8080` |

## Alert Annotations

The service supports the following custom annotations in AlertManager alerts:

| Annotation | Description |
|------------|-------------|
| `title` | Alert title |
| `description` | Alert description |
| `override_receivers` | Comma-separated list of chat IDs that replace default recipients |
| `additional_receivers` | Comma-separated list of chat IDs to add to default recipients |
| `do_not_send_resolved` | Set to "true" to suppress resolved notifications |
| `only_working_hours` | Set to "true" to send only during working hours (8:00-22:00) |

## Endpoints

- `/alert` - Webhook endpoint for AlertManager
- `/health-check` - Health check endpoint
- `/metrics` - Prometheus metrics endpoint

## Deployment

### Docker

```bash
docker run -p 8080:8080 \
  -e TELEGRAM_BOT_TOKEN=your_token \
  -e DEFAULT_RECEPIENTS=chat_id1,chat_id2 \
  -e GRAFANA_BASE_URL=https://grafana.example.com \
  -e PROM_BASE_URL=https://prometheus.example.com \
  -e AM_BASE_URL=https://alertmanager.example.com \
  ghcr.io/d13410n3/am-telegram:latest
```

## AlertManager Configuration Example

```yaml
receivers:
  - name: 'telegram'
    webhook_configs:
      - url: 'http://am-telegram:8080/alert'
        send_resolved: true

route:
  receiver: 'telegram'
  group_by: ['alertname', 'job']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
```

## Building from Source

```bash
go build -o am-telegram app.go
```

## License

This project is open source and available under the [MIT License](LICENSE).
