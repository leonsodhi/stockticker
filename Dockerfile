# Build the application from source
FROM golang:1.23.2 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY templates/ templates/
COPY Makefile Makefile

RUN make build


FROM ubuntu:noble

RUN apt-get update && apt-get --no-install-recommends -y install \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=build /app/bin/stockticker stockticker
COPY --from=build /app/templates templates

USER nobody
EXPOSE 8080
ENTRYPOINT ["/app/stockticker"]
