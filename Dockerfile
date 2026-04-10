ARG GO_VERSION=1.24
ARG ALPINE_VERSION=3.22

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS builder

WORKDIR /build
ADD go.mod go.sum ./

RUN go mod download -x

ADD . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -v -o agamennone ./cmd/agamennone

FROM alpine:${ALPINE_VERSION}
RUN apk add --no-cache ca-certificates py3-requests
WORKDIR /app
COPY --from=builder /build/agamennone /usr/local/bin/agamennone
ENTRYPOINT [ "agamennone" ]
