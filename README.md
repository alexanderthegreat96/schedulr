# Schedulr - A Modern Task Scheduler

Schedulr is a lightweight, crontab-inspired task scheduler that executes tasks based on JSON configurations. It supports shell commands and HTTP requests, running as a daemon with configurable execution intervals.

## Features

- **Simple Scheduling:**  
  Define tasks using JSON files, specifying delays and intervals.

- **Daemon Mode:**  
  Runs continuously in the background, managing task execution without user intervention.

- **Parallel Execution:**  
  Executes tasks asynchronously using a worker pool, allowing multiple tasks to run in parallel.

- **Interval-Based Execution:**  
  Schedule tasks to run at specified intervals (seconds, minutes, hours, etc.) based on your JSON configuration.

- **Dependency Management:**  
  Configure tasks to run dependencies using `RunBefore` and `RunAfter` fields, ensuring prerequisite tasks execute in order.

- **Comprehensive Logging:**  
  Logs events to the console and to dedicated log files for application and task events. Real-time log tailing and periodic log wiping features are also provided.

- **PID Management & Graceful Shutdown:**  
  Uses a PID file to prevent multiple instances and supports clean shutdown when receiving termination signals.

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.16 or later

### Installation

1. **Clone the repository:**

   ```bash
   git clone https://github.com/yourusername/schedulr.git
   cd schedulr
   ```

2. **Build the executable:**

   ```bash
   go build -o schedulr
   ```

3. **Create a configuration file:**

   Create a file (e.g., `schedulr.config`) in the root directory to define environment variables such as:

   ```
   SCHEDULR_DEV=1
   AppLogFilePath=/var/log/schedulr/app.log
   TasksLogFilePath=/var/log/schedulr/task.log
   ```

## Usage

### Running the Scheduler

To start Schedulr as a daemon, simply run:

```bash
./schedulr
```

This starts the scheduler loop, which continuously loads tasks from JSON files, schedules them based on their delay and interval configurations, and dispatches them for asynchronous execution.

### Adding Tasks

Use the `add` command to create a new task configuration. Supported task types are:

- `shell`: Executes a shell command.
- `http`: Executes an HTTP request.

Examples:

```bash
./schedulr add shell "Backup Database"
./schedulr add http "Ping API"
```

Task configurations are stored as JSON files in the designated tasks directory.

### Viewing and Managing Logs

Schedulr logs output to both the console and log files. A background log wiper function can be configured to truncate log files every x seconds to prevent indefinite growth.

#### Tailing Logs

Schedulr includes a built-in command to tail the latest log file in real time. For example, to tail the application logs:

```bash
./schedulr logs app
```

Or for task logs:

```bash
./schedulr logs task
```

The command finds the most recently modified log file in the appropriate directory and outputs new entries using the application's logging format.

## Command Line Usage

Schedulr uses a command-line interface built with Cobra. Below is an overview of the available commands:

### `add`

- **Usage:** `schedulr add [task_type] [task_name]`
- **Description:** Creates a new scheduled task configuration in JSON format.
- **Supported Task Types:**
  - `shell`: Executes a shell command.
  - `http`: Executes an HTTP request.

Examples:

```bash
schedulr add shell "Backup Database"
schedulr add http "Ping API"
```

### `logs`

- **Usage:** `schedulr logs [log_type]`
- **Description:** Tails the most recently modified log file in real time.
- **Supported Log Types:**
  - `app`: Application logs.
  - `task`: Task logs.

If no log type is specified, it defaults to `app`.

Examples:

```bash
schedulr logs app
schedulr logs task
```

### Default Behavior

Running `schedulr` without any arguments starts the scheduler daemon, which continuously processes and schedules tasks.

### Help Command

Use `schedulr help` or `schedulr --help` to view detailed information about all commands and options.

## Task Execution Details

### Scheduling

Each task is scheduled based on its delay and interval settings. Recurring tasks are automatically rescheduled.

### Dependencies

Tasks can be configured to run dependencies before or after the main task using `RunBefore` and `RunAfter` in the JSON configuration. The scheduler respects these dependencies by waiting for the dependent task’s scheduled time before executing.

### HTTP Task Execution

HTTP tasks support configurable methods, headers, and optional JSON bodies. Schedulr creates and sends HTTP requests, logs response status and body, and handles errors accordingly.

## Configuration

- **Task Directory:**  
  Task configurations are stored as JSON files under a specified directory, as defined in your configuration file.

- **Log File Paths:**  
  Set paths for your application and task log files using the `AppLogFilePath` and `TasksLogFilePath` environment variables.

- **Interval Settings:**  
  Control task execution timing via JSON fields for delay and interval (using units like seconds, minutes, and hours).


## Available Commands

| Command      | Description                                                  |
|--------------|--------------------------------------------------------------|
| `add`        | Create a new task configuration                              |
| `clear-logs` | Wipe log files                                               |
| `completion` | Generate the autocompletion script for the specified shell   |
| `help`       | Help about any command                                       |
| `list`       | Lists available tasks                                        |
| `logs`       | Tail the latest log file in real time                        |
| `run`        | Runs a specified task                                        |
| `setup`      | Creates necessary files and folders for Schedulr            |
| `start`      | Starts the scheduler daemon                                  |
| `status`     | Check if scheduler daemon is running                         |
| `stop`       | Stops the scheduler daemon                                   |


## Licence
MIT License

Copyright (c) [2025] [alexanderthegreat96]

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the “Software”), to deal
in the Software without restriction, including without limitation the rights  
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell  
copies of the Software, and to permit persons to whom the Software is  
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in  
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND, EXPRESS OR  
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,  
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE  
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER  
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,  
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN  
THE SOFTWARE.
