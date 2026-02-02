FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/ingestion ./services/ingestion/cmd/ingestion

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/ingestion /usr/local/bin/ingestion
EXPOSE 8083
ENTRYPOINT ["/usr/local/bin/ingestion"]
