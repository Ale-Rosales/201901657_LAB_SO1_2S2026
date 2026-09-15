FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o api3 main.go

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/api3 .
EXPOSE 8083
CMD ["./api3"]
