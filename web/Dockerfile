# syntax=docker/dockerfile:1

# Build the admin console image with web/ as the (small) build context. The
# app depends on the in-repo TypeScript SDK via a
# `file:../api/spec/packages/aip-client-javascript` dependency (outside the
# web/ context), so the SDK is injected as a named build context:
#
#   docker build -f web/Dockerfile \
#     --build-context openmeter-sdk=api/spec/packages/aip-client-javascript \
#     -t openmeter-admin web/
#
# Build stage: install deps with pnpm (via corepack, since Node 25+ no
# longer bundles it) and produce the static bundle in dist/.
FROM node:26-alpine AS build
WORKDIR /app

RUN npm install -g corepack \
    && corepack enable \
    && corepack prepare pnpm@11.1.2 --activate

# Place the SDK at the sibling path ../api/spec/... resolves to from /app.
# package.json + built dist/ are all consumers need (main/types point at
# dist/index.*).
COPY --from=openmeter-sdk package.json /api/spec/packages/aip-client-javascript/package.json
COPY --from=openmeter-sdk dist /api/spec/packages/aip-client-javascript/dist

# Install with a frozen lockfile first so dependency layers cache
# independently from source changes. pnpm-workspace.yaml is required here:
# it carries the pnpm v10+ build-script allowlist for esbuild.
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile

# Copy only what the build needs. A targeted copy keeps node_modules and
# other local state out of the image without requiring a .dockerignore.
COPY index.html vite.config.ts tsconfig.json tsconfig.app.json tsconfig.node.json ./
COPY public ./public
COPY src ./src

# Vite bakes VITE_* variables into the bundle at build time; expose them as
# build args so deployments can point the SPA at their Casdoor instance
# (defaults mirror .env.example).
ARG VITE_API_BASE=/api
ARG VITE_CASDOOR_ISSUER
ARG VITE_CASDOOR_CLIENT_ID
ARG VITE_CASDOOR_REDIRECT_URI
ARG VITE_CASDOOR_LOGOUT_REDIRECT_URI
ENV VITE_API_BASE=$VITE_API_BASE \
    VITE_CASDOOR_ISSUER=$VITE_CASDOOR_ISSUER \
    VITE_CASDOOR_CLIENT_ID=$VITE_CASDOOR_CLIENT_ID \
    VITE_CASDOOR_REDIRECT_URI=$VITE_CASDOOR_REDIRECT_URI \
    VITE_CASDOOR_LOGOUT_REDIRECT_URI=$VITE_CASDOOR_LOGOUT_REDIRECT_URI

RUN pnpm build

# Runtime stage: nginx serves the static bundle and proxies /api/ to the
# OpenMeter backend. OPENMETER_UPSTREAM is consumed by the nginx image's
# envsubst entrypoint when it renders nginx.conf.template into conf.d.
FROM nginx:alpine
ENV OPENMETER_UPSTREAM=openmeter:8888
COPY deploy/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
