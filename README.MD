# GoGraphSMTP

A lightweight SMTP server implemented in Go that forwards emails via the Microsoft Graph API.

## Features
- Supports both plain text and HTML emails.
- Handles email attachments.
- Logs all activities to a specified log file.

## Configuration
The server requires a configuration file in YAML format. Below is an example `config.yaml`:

```yaml
azure:
  client_id: "your-client-id"
  client_secret: "your-client-secret"
  tenant_id: "your-tenant-id"

smtp:
  address: ":25"
  domain: "localhost"

log_file: "/path/to/log/file.log"
