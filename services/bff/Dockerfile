FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/bff ./services/bff/cmd/bff

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/bff /usr/local/bin/bff
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/bff"]
