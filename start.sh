#!/bin/sh

# Start Nginx
nginx

# Start the Go application
./main &

# Keep the container running
tail -f /dev/null