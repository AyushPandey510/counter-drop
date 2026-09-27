FROM golang:1.27.1-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/counter-drop-api ./cmd/api

FROM alpine:3.22
RUN adduser -D -H counterdrop
WORKDIR /app
# The API applies SQL migrations from ./migrations at startup, so they must ship in the image.
COPY --from=build /src/migrations ./migrations
COPY --from=build /out/counter-drop-api /usr/local/bin/counter-drop-api
USER counterdrop
EXPOSE 8080
CMD ["counter-drop-api"]
