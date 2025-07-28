# --------------------
# STAGE 1: Base builder
# --------------------
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Required for Go modules + CGO
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOFLAGS="-mod=readonly"

# For CI: use Go proxy
ARG GOPROXY=https://proxy.golang.org
ENV GOPROXY=${GOPROXY}

# Copy only go.mod/sum to cache deps
COPY go.mod go.sum ./
RUN go mod download

# Now copy the source code
COPY . .

# Build control plane binary
RUN go build -o /control-plane ./cmd/control

# Build data plane binary
RUN go build -o /data-plane ./cmd/data

# --------------------
# STAGE 2: Final control plane image
# --------------------
FROM gcr.io/distroless/static-debian11 AS control-plane
COPY --from=builder /control-plane /app/control-plane
EXPOSE 50051 50052
ENTRYPOINT ["/app/control-plane"]

# --------------------
# STAGE 3: Final data plane image
# --------------------
FROM gcr.io/distroless/static-debian11 AS data-plane
COPY --from=builder /data-plane /app/data-plane
EXPOSE 60051
ENTRYPOINT ["/app/data-plane"]
