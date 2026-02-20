FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/analytics ./services/analytics/cmd/analytics

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/analytics /usr/local/bin/analytics
ENTRYPOINT ["/usr/local/bin/analytics"]
