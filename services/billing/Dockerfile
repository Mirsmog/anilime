FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /out/billing ./services/billing/cmd/billing

FROM alpine:3.19
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/billing /usr/local/bin/billing
EXPOSE 8087
ENTRYPOINT ["/usr/local/bin/billing"]
