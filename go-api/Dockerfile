FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN go mod tidy && CGO_ENABLED=0 go build -o /go-api .
FROM gcr.io/distroless/static
COPY --from=build /go-api /go-api
EXPOSE 8080
ENTRYPOINT ["/go-api"]
