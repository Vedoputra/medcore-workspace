# STAGE 1: Compilation Environment (Builder)
FROM golang:1.26-alpine3.23 AS builder

RUN apk add --no-cache ca-certificates=20261020-r0 git=2.52.0-r0 tzdata=2026e-r0

WORKDIR /src

# Cache dependency layer: salin go.mod dan go.sum terlebih dahulu
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /bin/medcore-billing-app main.go

# STAGE 2: Minimalist Production Runtime (distroless: tanpa shell dan package manager)
FROM gcr.io/distroless/static-debian12:nonroot AS runner

WORKDIR /app

COPY --from=builder /bin/medcore-billing-app /app/medcore-billing-app

# Unprivileged user (non-root), UID numerik
USER 10001:10001

# Exec form agar sinyal OS (SIGTERM) diterima biner langsung
ENTRYPOINT ["/app/medcore-billing-app"]
