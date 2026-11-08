# syntax=docker/dockerfile:1

FROM golang:1.27 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG GOAMD64=v2

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOAMD64=${GOAMD64} \
    go build -trimpath -ldflags="-s -w" -o /out/adminlte-demo .

FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=build /out/adminlte-demo /adminlte-demo

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/adminlte-demo"]
CMD ["-addr", ":8080", "-db", "/tmp/app.db"]
