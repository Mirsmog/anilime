FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/search ./services/search/cmd/search

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/search /usr/local/bin/search
EXPOSE 9094
ENTRYPOINT ["/usr/local/bin/search"]
