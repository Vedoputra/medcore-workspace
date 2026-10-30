FROM node:20-alpine3.19 AS builder
WORKDIR /usr/src/app
COPY package*.json tsconfig.json ./
RUN npm ci
COPY src/ ./src/
RUN npm run build && npm prune --omit=dev


FROM node:20-alpine3.19 AS runner
WORKDIR /app

# Hardening: patch OS, buang package manager (npm/yarn) dari image runtime
RUN apk upgrade --no-cache && \
    rm -rf /usr/local/lib/node_modules /opt/yarn-* \
           /usr/local/bin/npm /usr/local/bin/npx /usr/local/bin/yarn /usr/local/bin/yarnpkg

# Konfigurasi mode produksi
ENV NODE_ENV=production

# UID 1000 = user 'node' bawaan image Alpine resmi
USER 1000:1000
COPY --chown=node:node package*.json ./
COPY --chown=node:node --from=builder /usr/src/app/node_modules ./node_modules
COPY --chown=node:node --from=builder /usr/src/app/dist ./dist

EXPOSE 3000

CMD ["node", "dist/index.js"]