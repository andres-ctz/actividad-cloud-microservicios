FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o orders-api .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/orders-api .
ENV DB_PATH=/data/orders.db
RUN mkdir -p /data
VOLUME /data
EXPOSE 8002
CMD ["./orders-api"]
