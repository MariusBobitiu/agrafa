# syntax=docker/dockerfile:1.7

# Build static assets on the host builder architecture so multi-arch publishes
# don't need to run pnpm/vite under QEMU emulation.
FROM --platform=$BUILDPLATFORM node:22-alpine AS builder

WORKDIR /app
RUN corepack enable
RUN apk add --no-cache git

COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install

COPY . .

ARG VITE_API_URL=
ENV VITE_API_URL=$VITE_API_URL

RUN pnpm build

# Production stage
FROM nginx:alpine

COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
