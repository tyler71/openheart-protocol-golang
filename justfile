build:
  go build -ldflags "-s -w" -o openheart-protocol openheart.tylery.com/cmd/api
  du -hs openheart-protocol

