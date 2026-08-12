FROM golang:1.24-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /careflow-api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /careflow-migrate ./cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -o /careflow-seed ./cmd/seed

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
RUN addgroup -S careflow && adduser -S careflow -G careflow
WORKDIR /app
COPY --from=build /careflow-api /usr/local/bin/careflow-api
COPY --from=build /careflow-migrate /usr/local/bin/careflow-migrate
COPY --from=build /careflow-seed /usr/local/bin/careflow-seed
COPY migrations ./migrations
USER careflow
EXPOSE 8080
CMD ["careflow-api"]
