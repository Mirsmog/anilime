FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/catalog ./services/catalog/cmd/catalog

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/catalog /usr/local/bin/catalog
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/catalog"]
