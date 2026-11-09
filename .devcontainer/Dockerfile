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

RUN printf '%s\n' \
      '[user]' \
      '    name = AgungDH' \
      '    email = agungdh@live.com' \
      '[core]' \
      '    autocrlf = input' \
      '    excludesfile = /etc/gitignore_global' \
      > /etc/gitconfig.devcontainer

ENV GIT_CONFIG_GLOBAL=/etc/gitconfig.devcontainer

RUN go install github.com/pressly/goose/v3/cmd/goose@v3.29.0

RUN curl -fsSL https://opencode.ai/v2/install | bash -s -- --no-modify-path \
 && ln -s "$HOME/.opencode/bin/opencode" /usr/local/bin/opencode

ENV PATH="/go/bin:${PATH}"
