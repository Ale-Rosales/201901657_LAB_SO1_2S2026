FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY proto ./proto
COPY go-client ./go-client
RUN CGO_ENABLED=0 go build -o /out/go-client ./go-client

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/go-client /go-client
EXPOSE 8081
ENTRYPOINT ["/go-client"]
