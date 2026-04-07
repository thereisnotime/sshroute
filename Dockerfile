FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o sshroute .

FROM alpine:3.19
RUN apk add --no-cache openssh-client iputils
COPY --from=builder /app/sshroute /usr/local/bin/sshroute
ENTRYPOINT ["sshroute"]
