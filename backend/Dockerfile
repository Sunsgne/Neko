# syntax=docker/dockerfile:1
# Multi-stage build for the Neko control-plane API (static Go binary).
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/rosim ./cmd/rosim

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
COPY --from=build /out/rosim /app/rosim
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/api"]
