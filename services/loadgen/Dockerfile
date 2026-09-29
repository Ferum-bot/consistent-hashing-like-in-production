FROM golang:1.26-alpine AS build
WORKDIR /src
COPY libs/ libs/
COPY loadgen/go.mod loadgen/go.sum loadgen/
RUN cd loadgen && go mod download
COPY loadgen/ loadgen/
RUN cd loadgen && CGO_ENABLED=0 go build -o /out/loadgen ./cmd/service

FROM alpine:3.22
WORKDIR /app
COPY --from=build /out/loadgen /app/loadgen
COPY loadgen/config.yaml /app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/loadgen", "-config", "/app/config.yaml"]
