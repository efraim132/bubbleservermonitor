# Server Monitor

A simple terminal-based server monitoring tool built with Bubble Tea. Monitor multiple servers and track their online/offline status in real-time.

## Features

- Monitor multiple servers via TCP ping
- Real-time status updates every 5 seconds
- Color-coded status indicators (green for online, red for offline)
- Track how long servers have been offline
- Simple TUI (Terminal User Interface) for easy management

## Requirements

- Go 1.16 or higher
- Terminal with color support

## Installation

```bash
go build
```

## Usage

Run the application:

```bash
./bubbleservermonitor
```

### Keyboard Shortcuts

- **`a`** - Add a new server
- **`q`** or **`Ctrl+C`** - Quit the application

### Adding a Server

1. Press `a` to open the add server form
2. Enter the following information:
   - **Name**: A friendly name for the server
   - **IP Address**: The server's IP address or hostname
   - **Port**: The TCP port to monitor
3. Press `Enter` to submit the form
4. Press `Esc` to cancel

### Status Indicators

- **Green ●** - Server is online
- **Red ○** - Server is offline
- **Ping Since (s)** - Shows how many seconds the server has been offline (0 when online)

## How It Works

The tool pings each server every 5 seconds by attempting to establish a TCP connection. If the connection succeeds within 2 seconds, the server is marked as online. Otherwise, it's marked as offline and the offline timer starts counting.

## Debug Logs

Debug information is written to `debug.log` in the same directory as the application.
