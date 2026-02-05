FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/streaming-resolver ./services/streaming-resolver/cmd/streaming-resolver

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/streaming-resolver /usr/local/bin/streaming-resolver
EXPOSE 9095
ENTRYPOINT ["/usr/local/bin/streaming-resolver"]
