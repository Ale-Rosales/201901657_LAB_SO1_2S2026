FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o api2 main.go

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/api2 .
EXPOSE 8082
CMD ["./api2"]
