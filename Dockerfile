FROM golang:1.14.0 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build the app
#RUN go build -a -o /app/main
RUN go build -o /app/main

FROM gcr.io/distroless/base
COPY --from=builder /app/main /
COPY --from=builder /app/config.json /
EXPOSE 80
ENV VIRTUAL_PORT 80
ENV VIRTUAL_HOST apollo.localhost
ENV GIN_MODE release

COPY --from=wtfcoderz/static-healthcheck /healthcheck /
HEALTHCHECK --interval=10s --timeout=2s --start-period=1s --retries=2 CMD ["/healthcheck", "-tcp", "127.0.0.1:80"]

CMD ["/main"]
