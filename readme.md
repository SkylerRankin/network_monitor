# Network Monitor

![](./screenshots/screenshot_1.png)

A simple web app that displays a real-time graph of ping and network speed data. The server is designed to be a long-running service on a Linux host.

The server pings a few highly available DNS servers (Google, OpenDNS, and Cloudflare) for ping information, and uses a Go API for speedtest.net to get download/upload speed information.

## Build and run
```bash
# Create binary
make build

# Build and start server on :8080
make run
```

## Install as systemd service on Ubuntu

```bash
# Install the service
sudo make install

# Check the status
systemctl status netmon

# Uninstall the service
sudo make uninstall
```

The `install` target registers a systemd service. See `config/netmon.service` for the service configuration.
