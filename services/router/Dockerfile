FROM golang:1.26-alpine AS build
WORKDIR /src
COPY libs/ libs/
COPY router/go.mod router/go.sum router/
RUN cd router && go mod download
COPY router/ router/
RUN cd router && CGO_ENABLED=0 go build -o /out/router ./cmd/service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/router /app/router
COPY router/config.yaml /app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/router", "-config", "/app/config.yaml"]
