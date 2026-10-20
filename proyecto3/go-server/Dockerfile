FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY proto ./proto
COPY go-server ./go-server
RUN CGO_ENABLED=0 go build -o /out/go-server ./go-server

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/go-server /go-server
EXPOSE 50051
ENTRYPOINT ["/go-server"]
