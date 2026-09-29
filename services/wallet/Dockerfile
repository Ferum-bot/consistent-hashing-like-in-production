FROM golang:1.26-alpine AS build
WORKDIR /src
COPY libs/ libs/
COPY wallet/go.mod wallet/go.sum wallet/
RUN cd wallet && go mod download
COPY wallet/ wallet/
RUN cd wallet && CGO_ENABLED=0 go build -o /out/wallet ./cmd/service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/wallet /app/wallet
COPY wallet/config.yaml /app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/wallet", "-config", "/app/config.yaml"]
