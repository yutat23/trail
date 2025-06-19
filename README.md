# Trail

A robust file tailing utility written in Go that provides enhanced functionality for monitoring log files with automatic log rotation support.

## Features

- **File Tailing**: Monitor individual files with real-time output
- **Directory Monitoring**: Automatically tail the latest file in a directory
- **Log Rotation Support**: Seamlessly follows files even when they are rotated
- **Configurable**: Customizable options for different use cases

## Installation

### From Source

```bash
git clone https://github.com/yutat23/trail
cd trail
go build -o trail main.go
```

## Usage

```
trail <command> [options] <path>
```

### Commands

- `file` or `-f`: Tail a specific file and follow it
- `dir` or `-d`: Tail the latest file in a directory and follow it
- `help`, `-h`, or `--help`: Show help message

### File Mode

Monitor a specific file:

```bash
trail file [options] <file_path>
```

#### Options

- `-n <N>`: Print last N lines before following (default: 10)

#### Examples

```bash
# Tail the last 10 lines of app.log and follow
trail file app.log

# Tail the last 100 lines of app.log and follow
trail file -n 100 app.log

# On Windows
trail.exe file "C:\Logs\application.log"
```

### Directory Mode

Monitor the latest file in a directory:

```bash
trail dir [options] <directory_path>
```

#### Options

- `-interval <duration>`: Polling fallback interval (default: 5s)

#### Examples

```bash
# Monitor the latest file in the logs directory
trail dir ./logs

# Monitor with custom polling interval
trail dir -interval 10s ./logs

# On Windows
trail.exe dir "C:\Logs\MyService"
```

## How It Works

### File Mode
- Reads and displays the last N lines of the specified file
- Continuously monitors the file for new content
- Handles file rotation by reopening the file when necessary

### Directory Mode
- Scans the directory to find the file with the latest modification time
- Monitors the directory for new files
- Automatically switches to newer files when they appear
- Uses filesystem notifications with polling fallback for reliability

## Dependencies

- [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file system notifications
- [tail](https://github.com/hpcloud/tail) - File tailing library with rotation support

## Requirements

- Go 1.24.4 or later
