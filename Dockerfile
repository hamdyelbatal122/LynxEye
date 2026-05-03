FROM golang:1.23 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/lynxeye ./cmd/lynxeye

FROM gcr.io/distroless/static-debian12
WORKDIR /app

COPY --from=builder /out/lynxeye /usr/local/bin/lynxeye
COPY config.example.yaml /app/config.example.yaml

ENTRYPOINT ["/usr/local/bin/lynxeye"]
