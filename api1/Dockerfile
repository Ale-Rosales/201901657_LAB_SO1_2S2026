FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY main.go ./
RUN go build -o api1 main.go

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/api1 .
EXPOSE 8081
CMD ["./api1"]
