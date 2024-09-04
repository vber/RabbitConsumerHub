#!/bin/sh

# Start the Go application
./main &

# Wait for the Go application to start
sleep 5

# Start Nginx
nginx

# Keep the container running
tail -f /dev/null