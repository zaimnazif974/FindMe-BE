FROM golang:1.25-alpine AS build
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/findme-api ./cmd/api

FROM alpine:3.21
RUN apk add --no-cache ca-certificates && addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=build /out/findme-api ./findme-api
COPY migrations ./migrations
USER app
EXPOSE 15007
ENTRYPOINT ["./findme-api"]
