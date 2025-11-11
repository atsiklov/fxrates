# syntax=docker/dockerfile:1.7

FROM golang:1.24-alpine AS build-stage
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build  -o ./img1-build-dir ./cmd

FROM gcr.io/distroless/base-debian12 AS build-release-stage
WORKDIR /app

COPY --from=build-stage /app/img1-build-dir /app/img2-build-dir
COPY config.yaml /app/config.yaml
EXPOSE 8080
ENTRYPOINT ["/app/img2-build-dir"]