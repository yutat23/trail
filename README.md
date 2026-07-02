# Trail

```
████████╗██████╗  █████╗ ██╗██╗     
╚══██╔══╝██╔══██╗██╔══██╗██║██║     
   ██║   ██████╔╝███████║██║██║     
   ██║   ██╔══██╗██╔══██║██║██║     
   ██║   ██║  ██║██║  ██║██║███████╗
   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝╚══════╝
```

A robust file tailing utility written in Go that provides enhanced functionality for monitoring log files with automatic log rotation support and colored output.


[trail.webm](https://github.com/user-attachments/assets/35fb14cc-0d69-436e-b30e-6f21eac73f97)

## Features

- **File Tailing**: Monitor individual files with real-time output
- **Directory Monitoring**: Automatically tail the latest file in a directory
- **Pattern Matching**: Support for wildcard patterns to filter files (e.g., `*.log`, `app-*.log`)
- **Log Rotation Support**: Seamlessly follows files even when they are rotated
- **Colored Output**: Highlight specific patterns with custom colors using regular expressions
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
trail [global options] <command> [options] <path>
```

### Global Options

- `-h`, `--help`: Show help message
- `-v`, `--version`: Show version
- `--no-logo`: Disable logo display
- `--no-color-logo`: Disable colored logo
- `--color <mode>`: Color output mode: `auto`, `always`, or `never` (default: `auto`)

### Commands

- `file` or `-f`: Tail a specific file and follow it
- `dir` or `-d`: Tail the latest file in a directory
- `help`, `-h`, or `--help`: Show help message

### File Mode

Monitor a specific file:

```bash
trail file [options] <file_path>
```

#### Options

- `-n <N>`: Print last N lines before following (default: 10)
- `-c <pattern>`: Color pattern in format 'color:regex' (can be used multiple times)

#### Color Options

Available colors:
- Basic colors: `red`, `green`, `blue`, `yellow`, `magenta`, `cyan`, `white`, `black`
- Bright colors: `brightred`, `brightgreen`, `brightblue`, `brightyellow`, `brightmagenta`, `brightcyan`, `brightwhite`

Color pattern format: `color:regex`
- Prefer repeating `-c` for multiple patterns, especially when the regex itself contains commas
- Comma-separated color entries are also supported when each entry starts with `color:`
- If matches overlap, later color patterns take precedence
- Example: `"red:ERROR,green:DEBUG,yellow:WARN"`

#### Examples

```bash
# Tail the last 10 lines of app.log and follow
trail file app.log

# Tail the last 100 lines of app.log and follow
trail file -n 100 app.log

# Highlight ERROR in red, DEBUG in green, WARN in yellow
trail file -c "red:ERROR,green:DEBUG,yellow:WARN" app.log

# Highlight date patterns in blue
trail file -c "blue:\d{2}-\d{2}" app.log

# Highlight a regex that contains a comma
trail file -c "red:\d{2,4}" app.log

# Multiple color patterns
trail file -c "red:ERROR,green:DEBUG,blue:\d{4}-\d{2}-\d{2}" app.log
trail file -c "red:ERROR" -c "green:DEBUG" app.log

# Force ANSI color output even when stdout is redirected
trail --color always file -c "red:ERROR" app.log

# On Windows
trail.exe file "C:\Logs\application.log"
trail.exe file -c "red:ERROR,green:DEBUG" "C:\Logs\application.log"
```

### Directory Mode

Monitor the latest file in a directory:

```bash
trail dir [options] <directory_path>
```

#### Options

- `-n <N>`: Print last N lines before following (default: 10)
- `-interval <duration>`: Polling fallback interval (default: 5s)
- `-c <pattern>`: Color pattern in format 'color:regex' (can be used multiple times)
- `-pattern <pattern>`: File pattern to match (e.g., `*.log`, `app-*.log`, `service-*.txt`)

#### Pattern Matching

The `-pattern` option allows you to specify which files to monitor using wildcard patterns:

- `*.log` - Monitor all files with `.log` extension
- `app-*.log` - Monitor files starting with "app-" and ending with ".log"
- `service-*.txt` - Monitor files starting with "service-" and ending with ".txt"
- `*` - Monitor all files (default behavior)

#### Examples

```bash
# Monitor the latest file in the logs directory
trail dir ./logs

# Monitor with last 20 lines displayed first
trail dir -n 20 ./logs

# Monitor only .log files in the directory
trail dir -pattern "*.log" ./logs

# Monitor files with specific naming pattern
trail dir -pattern "app-*.log" ./logs
trail dir -pattern "service-*.txt" ./logs

# Monitor with custom polling interval
trail dir -interval 10s ./logs

# Monitor with colored output
trail dir -c "red:ERROR,green:DEBUG,yellow:WARN" ./logs

# Combine pattern matching with colored output
trail dir -pattern "*.log" -c "red:ERROR,green:DEBUG,yellow:WARN" ./logs

# Combine all options: pattern, last N lines, and colored output
trail dir -pattern "app-*.log" -n 50 -c "red:ERROR" ./logs

# On Windows
trail.exe dir "C:\Logs\MyService"
trail.exe dir -n 20 "C:\Logs\MyService"
trail.exe dir -pattern "*.log" "C:\Logs\MyService"
trail.exe dir -c "yellow:WARN,red:ERROR" "C:\Logs\MyService"
trail.exe dir -pattern "app-*.log" -n 50 -c "red:ERROR" "C:\Logs\MyService"
```

## How It Works

### File Mode
- Reads and displays the last N lines of the specified file
- Continuously monitors the file for new content
- Handles file rotation by reopening the file when necessary
- Applies color highlighting to matching patterns in real-time

### Directory Mode
- Scans the directory to find the file with the latest modification time
- Supports wildcard pattern matching to filter files (e.g., only `.log` files)
- Monitors the directory for new files in that directory
- Automatically switches to newer files when they appear
- Uses filesystem notifications with interval polling fallback for directory changes
- Applies color highlighting to all monitored files

### Color Highlighting
- Uses regular expressions to match patterns in log lines
- Supports multiple color patterns simultaneously
- Processes patterns in order, with later patterns taking precedence
- Respects terminal color detection by default; use `--color always` to force ANSI color output
- Works with both file and directory monitoring modes

## Dependencies

- [fsnotify](https://github.com/fsnotify/fsnotify) - Cross-platform file system notifications
- [tail](https://github.com/nxadm/tail) - File tailing library with rotation support
- [color](https://github.com/fatih/color) - Colored terminal output

## Requirements

- Go 1.24.4 or later
