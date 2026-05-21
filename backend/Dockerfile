FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN GOFLAGS="-mod=mod" GONOSUMDB="*" go build -o dockyard .

FROM alpine:3.23.3
RUN apk upgrade --no-cache && apk --no-cache add ca-certificates
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
WORKDIR /home/appuser/
COPY --from=builder /app/dockyard .
RUN chown appuser:appgroup dockyard
USER appuser
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget -qO- http://localhost:8081/health || exit 1
EXPOSE 8081
CMD ["./dockyard"]
