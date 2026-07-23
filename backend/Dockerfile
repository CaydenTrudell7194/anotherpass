FROM golang:1.21-alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /build
COPY go.mod go.sum* ./
RUN go mod tidy 2>/dev/null || go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o backend ./cmd/

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata sqlite-libs
WORKDIR /app
COPY --from=builder /build/backend .
EXPOSE 18888
CMD ["./backend"]
