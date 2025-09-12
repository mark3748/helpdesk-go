# Grafana Integration

The API exposes metric endpoints that can be visualized in Grafana using the JSON API data source.

## Endpoints

- `GET /metrics/dashboard` – combined SLA attainment, average resolution time, and ticket volume.
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
    {"type": "stat", "title": "SLA Attainment", "targets": [{"method": "GET", "url": "/metrics/dashboard", "dataPath": "sla.sla_attainment"}]},
    {"type": "stat", "title": "Avg Resolution (ms)", "targets": [{"method": "GET", "url": "/metrics/dashboard", "dataPath": "avg_resolution_ms"}]},
    {"type": "timeseries", "title": "Ticket Volume", "targets": [{"method": "GET", "url": "/metrics/dashboard", "dataPath": "volume"}]}
  ]
}
```

Use Grafana's panel options to map fields from the JSON responses to visualizations.

## Prometheus Queries

If scraping the `/metrics` endpoint with Prometheus, these queries provide similar insights:

```promql
# Tickets created per day
increase(tickets_created_total[1d])

# SLA attainment (requires counters `tickets_sla_met_total` and `tickets_resolved_total`)
increase(tickets_sla_met_total[1d]) / increase(tickets_resolved_total[1d])

# Average resolution time
increase(ticket_resolution_ms_sum[1d]) / increase(ticket_resolution_ms_count[1d])
```
