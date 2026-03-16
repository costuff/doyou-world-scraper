# Build stage
FROM node:20-alpine AS builder

WORKDIR /build

# Copy package files
COPY package.json package-lock.json ./
RUN npm ci

# Copy source
COPY . .

# Build the app — VITE_ANALYTICS_API_URL baked in at build time (proxied by nginx in prod, by vite in dev)
# CI uses the ARG default; override with --build-arg if needed
ARG VITE_ANALYTICS_API_URL=/analytics-api
ENV VITE_ANALYTICS_API_URL=$VITE_ANALYTICS_API_URL
RUN npm run build

# Production stage
FROM nginx:alpine

# Copy custom nginx config
COPY <<EOF /etc/nginx/conf.d/default.conf
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    resolver 127.0.0.11 ipv6=off;

    # API proxy — use full container name to avoid DNS ambiguity when
    # frontend is on multiple Docker networks (scraper + analytics)
    location /api/ {
        proxy_pass http://doyou-api:3001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    # Analytics API proxy (separate analytics service, direct proxy_pass to avoid variable path issues)
    location /analytics-api/ {
        proxy_pass http://doyou-analytics-api:8080/;
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_connect_timeout 2s;
        proxy_read_timeout 15s;
    }

    # Frontend routes
    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml application/xml+rss text/javascript;
}
EOF

# Copy built app from builder
COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
