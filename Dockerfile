FROM golang:1.25.3
ARG BINARY_NAME=mr
WORKDIR /app
COPY . .
RUN go build -o "${BINARY_NAME}"
