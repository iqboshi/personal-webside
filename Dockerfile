FROM golang:1.26-alpine AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN go build -o /out/personal-webside ./cmd/server

FROM alpine:3.22

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S app \
  && adduser -S app -G app \
  && mkdir -p /app/data \
  && chown -R app:app /app

COPY --from=build /out/personal-webside /app/personal-webside
COPY content ./content

ENV ADDR=:8080
ENV DB_PATH=/app/data/homepage.db
ENV CONTENT_DIR=/app/content/articles
ENV STATIC_DIR=/app/frontend/dist

USER app

EXPOSE 8080

CMD ["/app/personal-webside"]
