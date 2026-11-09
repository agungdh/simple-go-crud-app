# syntax=docker/dockerfile:1

FROM golang:1.27

COPY gitignore_global /etc/gitignore_global

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      jq \
      make \
      sqlite3 \
 && rm -rf /var/lib/apt/lists/*

RUN git config --system user.name "AgungDH" \
 && git config --system user.email "agungdh@live.com" \
 && git config --system core.autocrlf input \
 && git config --system core.excludesfile /etc/gitignore_global

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.29.0

RUN curl -fsSL https://opencode.ai/v2/install | bash -s -- --no-modify-path \
 && ln -s "$HOME/.opencode/bin/opencode" /usr/local/bin/opencode

ENV PATH="/go/bin:${PATH}"
