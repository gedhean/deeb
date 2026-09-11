# Multi-stage Dockerfile for Deeb Frontend
# Stage 1: Build the React application
FROM node:24-alpine AS builder

# Build argument for API base URL — REQUIRED per environment; staging must
# pass --build-arg VITE_API_BASE_URL=https://deeb-staging.nsx.services.
# Production URL is the default only when the arg is omitted (explicitly
# documented foot-gun); staging CI must always pass it.
ARG VITE_API_BASE_URL=https://deeb.nsx.services

# Install pnpm — pin to 11.9.0 to match CI (pnpm_action-setup) and local dev.
# pnpm 10+ reads build-script approvals from pnpm-workspace.yaml (allowBuilds:
# esbuild/msw); pnpm 9 instead treats pnpm-workspace.yaml as a workspace def and
# fails with "packages field missing or empty" on `pnpm run`. The lockfile
# (lockfileVersion 9.0) is read unchanged by pnpm 11.
RUN corepack enable && corepack prepare pnpm@11.9.0 --activate

# Set working directory
WORKDIR /app

# Copy package files. pnpm-workspace.yaml carries the build-script approvals
# (allowBuilds: esbuild/msw); without it here, `pnpm install` aborts with
# ERR_PNPM_IGNORED_BUILDS.
COPY frontend/package.json frontend/pnpm-lock.yaml frontend/.npmrc frontend/pnpm-workspace.yaml ./

# Install dependencies
RUN pnpm install --frozen-lockfile

# Copy backend IAM config for RBAC generation
COPY backend/config/iam /backend/config/iam

# Copy source code
COPY frontend/ .

# Build the application with API base URL
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}
RUN pnpm run build

# Stage 2: Serve with nginx
FROM nginx:alpine

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy custom nginx config
COPY <<'EOF' /etc/nginx/conf.d/default.conf
server {
    listen 8080;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_types text/plain text/css text/xml text/javascript application/x-javascript application/xml+rss application/json application/javascript;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # SPA routing - serve index.html for all routes
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # Health check endpoint
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
EOF

# Copy built files from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Use existing nginx user, fix permissions
RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid

# Switch to non-root user
USER nginx

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Start nginx
CMD ["nginx", "-g", "daemon off;"]
