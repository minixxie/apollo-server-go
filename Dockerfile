FROM golang:1.15 as build

WORKDIR /app

COPY . .

RUN cd cmd/mock-apollo-go && go build -o /go/bin/app

FROM gcr.io/distroless/base
COPY --from=build /go/bin/app /
CMD ["/app"]