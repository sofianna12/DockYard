FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod ./
COPY . .
RUN GOFLAGS="-mod=mod" GONOSUMDB="*" go build -o dockyard .

FROM alpine:latest
RUN apk upgrade --no-cache && apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/dockyard .
EXPOSE 8081
CMD ["./dockyard"]
