FROM golang:1.25.3
ARG BINARY_NAME=csh
WORKDIR /app
COPY src/. .
RUN go build -o "${BINARY_NAME}"
