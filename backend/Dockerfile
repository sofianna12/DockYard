FROM golang:1.26-alpine@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN GOFLAGS="-mod=mod" GONOSUMDB="*" go build -o dockyard .

FROM alpine:3.23.3@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659
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
