FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN GOOS=linux go build -o /shigure-entry ./main.go

#--

FROM ubuntu:24.04
WORKDIR /
COPY --from=builder /shigure-entry /shigure-entry
ENTRYPOINT [ "/shigure-entry" ]