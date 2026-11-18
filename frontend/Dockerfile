# Build stage runs on the native build platform; the output is static, architecture-independent assets.
FROM --platform=$BUILDPLATFORM node:20-alpine AS builder

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

# Dynamic upstream resolution requires nginx 1.27.3 or newer.
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy built app from builder
COPY --from=builder /build/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
