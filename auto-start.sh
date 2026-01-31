#!/bin/bash
# 1. Get the script's actual directory
DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)
cd "$DIR" || { echo "Error: Could not change directory"; exit 1; }

# 2. Define Log File
LOG_FILE="schedulr_autostart.log"

# 3. THE MAGIC LINE: Redirect all output (stdout and stderr)
# to both the terminal and the log file.
exec > >(tee "$LOG_FILE") 2>&1

echo "---------------------------------------"
echo "Timestamp: $(date)"
echo "Schedulr Auto-Start Script Initialized"
echo "Working Directory: $DIR"

# Remove PID if it exists
if [ -f "schedulr.pid" ]; then
    echo "Found old schedulr.pid, removing..."
    rm -f "schedulr.pid"
else
    echo "No old PID file found. Clean start."
fi

# Run the daemon
echo "Attempting to start schedulr..."
./schedulr start

# Check if it actually started
sleep 1
if pgrep -f "schedulr" > /dev/null; then
    echo "SUCCESS: Schedulr is running!"
else
    echo "FAILED: Schedulr did not start. Check permissions of the binary."
fi
echo "---------------------------------------"
		