# Grafana Integration

The API exposes metric endpoints that can be visualized in Grafana using the JSON API data source.

## Endpoints

- `GET /metrics/sla` – percentage of tickets resolved within their SLA.
- `GET /metrics/resolution` – average resolution time in milliseconds for resolved tickets.
- `GET /metrics/tickets` – ticket volume per day for the last 30 days.

## Configuring Grafana

1. Install the [JSON API data source](https://grafana.com/grafana/plugins/simpod-json-datasource/).
2. Add a new data source pointing at your Helpdesk API base URL.
3. Create panels that query the endpoints above. Each panel should use the appropriate HTTP method and URL.

## Sample Dashboard

```json
{
  "panels": [
    {"type": "stat", "title": "SLA Attainment", "targets": [{"method": "GET", "url": "/metrics/sla"}]},
    {"type": "stat", "title": "Avg Resolution (ms)", "targets": [{"method": "GET", "url": "/metrics/resolution"}]},
    {"type": "timeseries", "title": "Ticket Volume", "targets": [{"method": "GET", "url": "/metrics/tickets"}]}
  ]
}
```

Use Grafana's panel options to map fields from the JSON responses to visualizations.
