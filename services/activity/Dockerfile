FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/activity ./services/activity/cmd/activity

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/activity /usr/local/bin/activity
EXPOSE 8085
ENTRYPOINT ["/usr/local/bin/activity"]
