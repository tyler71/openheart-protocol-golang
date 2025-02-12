FROM golang:1.23.5 AS build
LABEL authors="Tyler Yeager"

WORKDIR /build

COPY . .
RUN go mod tidy \
 && ./build.sh linux/amd64 \
 && gzip -d build/openheart-protocol-linux-amd64.gz \
 && chmod +x build/openheart-protocol-linux-amd64 \
 && mv build/openheart-protocol-linux-amd64 openheart-protocol



FROM debian:12-slim AS prod

WORKDIR /app
COPY --from=build /build/openheart-protocol /app/openheart-protocol

ENTRYPOINT ["/app/openheart-protocol"]