# Build Image
FROM golang:1.18.3 AS build

# Create application directory
RUN mkdir -p /go/src/app
WORKDIR /go/src/app

COPY src/go.mod src/go.sum /go/src/app/
RUN go mod download

ADD ./src /go/src/app
RUN go build -o /go/bin/app

FROM gcr.io/distroless/base-debian11:nonroot

COPY --from=build /go/bin/app /

CMD ["/app"]
