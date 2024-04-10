FROM golang:alpine as builder
WORKDIR /src/app
COPY go.mod go.sum main.go ./
RUN go build -o server

FROM alpine
WORKDIR /root/
COPY --from=builder /src/app ./app
CMD ["./app/server"]
