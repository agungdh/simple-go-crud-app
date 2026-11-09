# syntax=docker/dockerfile:1

FROM golang:1.27

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      jq \
      make \
      sqlite3 \
 && rm -rf /var/lib/apt/lists/*

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.29.0

ENV PATH="/go/bin:${PATH}"
