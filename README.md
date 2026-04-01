# Stockyard Dossier

**CRM for solo founders and consultants — contacts, notes, follow-up reminders, deal pipeline**

Part of the [Stockyard](https://stockyard.dev) family of self-hosted developer tools.

## Quick Start

```bash
docker run -p 9280:9280 -v dossier_data:/data ghcr.io/stockyard-dev/stockyard-dossier
```

Or with docker-compose:

```bash
docker-compose up -d
```

Open `http://localhost:9280` in your browser.

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `9280` | HTTP port |
| `DATA_DIR` | `./data` | SQLite database directory |
| `DOSSIER_LICENSE_KEY` | *(empty)* | Pro license key |

## Free vs Pro

| | Free | Pro |
|-|------|-----|
| Limits | 50 contacts, no pipeline | Unlimited contacts and pipeline |
| Price | Free | $2.99/mo |

Get a Pro license at [stockyard.dev/tools/](https://stockyard.dev/tools/).

## Category

Creator & Small Business

## License

Apache 2.0
