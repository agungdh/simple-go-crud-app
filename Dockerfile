# syntax=docker/dockerfile:1

# ---------------------------------------------------------------------------
# Build stage (Debian 13 trixie)
#
# modernc.org/sqlite is pure Go, so a static binary is enough; no CGO or
# libsqlite3 in the runtime image.
# ---------------------------------------------------------------------------
FROM golang:1.27.4-trixie AS build

WORKDIR /src

# Download modules first so the layer is reused while source code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-s -w" \
        -o /out/simple-go-crud-app .

# Prepare the database directory with the distroless nonroot UID/GID (65532)
# so the volume is writable by the runtime user.
RUN mkdir -p /out/data \
    && chown 65532:65532 /out/data

# ---------------------------------------------------------------------------
# Runtime stage (Debian 13 distroless)
#
# No shell and no package manager: the image only contains the static binary,
# CA certificates, tzdata, and the nonroot user from the distroless base.
# ---------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build --chown=65532:65532 /out/data /data
COPY --from=build /out/simple-go-crud-app /usr/local/bin/simple-go-crud-app

WORKDIR /app

# The SQLite database lives on a volume so migrations and data survive
# container restarts. Mount it with owner 65532:65532 (or use the anonymous
# volume created from this image).
ENV ADDR=:8080 \
    DB_PATH=/data/app.db \
    CORS_ORIGINS=http://localhost:8082

VOLUME ["/data"]
EXPOSE 8080

USER 65532:65532

ENTRYPOINT ["/usr/local/bin/simple-go-crud-app"]
