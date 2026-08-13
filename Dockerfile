# Build
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY internal ./internal
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /solaris-api ./cmd/server

# Run
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /solaris-api /app/solaris-api
COPY index.html logo.png /app/
ENV PORT=8080
ENV STATIC_DIR=/app
EXPOSE 8080
CMD ["/app/solaris-api"]
