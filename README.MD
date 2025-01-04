# GoGraphSMTP

A lightweight SMTP server implemented in Go that forwards emails via the Microsoft Graph API.

## Features
- Supports both plain text and HTML emails.
- Handles email attachments.
- Logs all activities to a specified log file.

## Configuration
The server requires a configuration file in YAML format. Below is an example `GoSMTP.yaml`:

```yaml
azure:
  client_id: "your-client-id"
  client_secret: "your-client-secret"
  tenant_id: "your-tenant-id"

smtp:
  address: ":25"
  domain: "localhost"

log_file: "/path/to/log/file.log"
```

## Setup

### 1. Build the Project
1. Clone the repository:
   ```bash
   git clone <repository_url>
   cd smtp_graph_project
   ```
2. Install dependencies:
   ```bash
   go mod tidy
   ```
3. Build the project:
   ```bash
   go build -o GoGraphSMTP main.go
   ```

### 2. Deploy as a Systemd Service

#### Service File Configuration
Create a systemd service file `gographsmtp.service` with the following content:

```ini
[Unit]
Description=GoGraphSmtp SMTP to Graph API Bridge
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/GoGraphSMTP
Restart=always
RestartSec=10
WorkingDirectory=/etc/GoGraphSMTP

[Install]
WantedBy=multi-user.target
```

#### Deployment Steps
Follow these steps to deploy the service:

1. Copy the binary file:
   ```bash
   cp GoGraphSMTP /usr/local/bin/
   ```

2. Create a directory for configuration:
   ```bash
   mkdir -p /etc/GoGraphSMTP
   ```

3. Copy the configuration file:
   ```bash
   cp GoSMTP.yaml /etc/GoGraphSMTP/
   ```

4. Copy the systemd service file:
   ```bash
   cp gographsmtp.service /etc/systemd/system/
   ```

5. Reload the systemd configuration:
   ```bash
   systemctl daemon-reload
   ```

6. Start the service:
   ```bash
   systemctl start gographsmtp
   ```

7. Enable the service to start on boot:
   ```bash
   systemctl enable gographsmtp
   ```

8. Check the service status:
   ```bash
   systemctl status gographsmtp
   ```

### 3. Managing the Service

Use the following commands to manage the service:

- Stop the service:
  ```bash
  systemctl stop gographsmtp
  ```

- Restart the service:
  ```bash
  systemctl restart gographsmtp
  ```

- View service logs:
  ```bash
  journalctl -u gographsmtp
  ```

## Testing
1. Create a test PHP script or use any SMTP client to test the service.
2. Verify logs in the file specified in the `log_file` configuration or with `journalctl`.

## Notes
- Ensure that your Azure app has the `Mail.Send` permission.
- Test the service with a sample email to confirm proper configuration.

## License
MIT License
