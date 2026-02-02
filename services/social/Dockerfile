FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/social ./services/social/cmd/social

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/social /usr/local/bin/social
EXPOSE 8086
ENTRYPOINT ["/usr/local/bin/social"]
