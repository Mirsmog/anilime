FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/streaming ./services/streaming/cmd/streaming

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/streaming /usr/local/bin/streaming
EXPOSE 8084
ENTRYPOINT ["/usr/local/bin/streaming"]
