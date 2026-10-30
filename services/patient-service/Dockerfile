# ==============================================================================
# STAGE 1: Build & Transpilation Stage
# ==============================================================================
FROM node:20-alpine3.19 AS builder

WORKDIR /usr/src/app

# Salin package definitions untuk memanfaatkan Docker Layer Caching
COPY package*.json tsconfig.json ./

# Pasang seluruh dependensi (termasuk devDependencies untuk kompilasi TypeScript)
RUN npm ci

# Salin source code TypeScript
COPY src/ ./src/

# Transpilasi TypeScript ke JavaScript murni (output: /usr/src/app/dist)
RUN npm run build

# Bersihkan devDependencies agar hanya menyisakan runtime dependencies
RUN npm prune --omit=dev

# ==============================================================================
# STAGE 2: Production Distroless/Alpine Runner
# ==============================================================================
FROM node:20-alpine3.19 AS runner

WORKDIR /app

# Konfigurasi mode produksi
ENV NODE_ENV=production

# Gunakan user non-root 'node' bawaan image Alpine resmi
USER node

# Salin modul produksi dan hasil transpilasi biner JS dari builder
COPY --chown=node:node package*.json ./
COPY --chown=node:node --from=builder /usr/src/app/node_modules ./node_modules
COPY --chown=node:node --from=builder /usr/src/app/dist ./dist

EXPOSE 3000

CMD ["node", "dist/index.js"]