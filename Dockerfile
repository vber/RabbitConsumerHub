# Build stage for Go backend
FROM golang:1.21-alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Install build dependencies
RUN apk add --no-cache gcc musl-dev
# Build with CGO enabled
RUN CGO_ENABLED=1 GOOS=linux go build -o main .

# Build stage for React frontend
FROM node:18-alpine AS frontend-builder
WORKDIR /app
COPY management_ui/package.json management_ui/package-lock.json ./
RUN npm ci
COPY management_ui ./
RUN npm run build

# Final stage
FROM alpine:3.18
WORKDIR /app

# Copy Go binary from backend-builder
COPY --from=backend-builder /app/main .

# Install runtime dependencies
RUN apk add --no-cache libc6-compat

# Install Nginx
RUN apk add --no-cache nginx

# Copy React build from frontend-builder
COPY --from=frontend-builder /app/build /usr/share/nginx/html

# Copy necessary files
COPY management_ui/src/locales /usr/share/nginx/html/locales
COPY README.md .

# Create logs directory
RUN mkdir logs

# Install SQLite
RUN apk add --no-cache sqlite

# Copy Nginx configuration
COPY nginx.conf /etc/nginx/nginx.conf

# Expose ports
EXPOSE 1981 80

# Copy start script
COPY start.sh .
RUN chmod +x start.sh

# Use the start script as the entry point
CMD ["./start.sh"]
