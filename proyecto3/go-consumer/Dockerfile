FROM golang:1.25 AS build
ENV GOTOOLCHAIN=auto
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY proto ./proto
COPY go-consumer ./go-consumer
RUN CGO_ENABLED=0 go build -o /out/go-consumer ./go-consumer

FROM gcr.io/distroless/static-debian12
COPY --from=build /out/go-consumer /go-consumer
ENTRYPOINT ["/go-consumer"]
