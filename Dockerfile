# # from golang:1.24
# FROM golang:1.24 AS builder
# # Set the working directory
# WORKDIR /
# COPY ["/", "/"]
# RUN go mod download
# RUN go build -o kokaq-server ./cmd/kokaq-server/main.go

# FROM debian:bookworm-slim
# WORKDIR /root/
# COPY --from=builder /kokaq-server /root/kokaq-server
# EXPOSE 4242
# CMD ["./kokaq-server", "-c", "/root/kokaq-server/config.yaml"]