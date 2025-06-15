# # from golang:1.24
# FROM golang:1.24 AS builder
# # Set the working directory
# WORKDIR /
# COPY ["/", "/"]
# RUN go mod download
# RUN go build -o server ./cmd/server/main.go

# FROM debian:bookworm-slim
# WORKDIR /root/
# COPY --from=builder /server /root/server
# EXPOSE 4242
# CMD ["./server", "-c", "/root/server/config.yaml"]