# syntax=docker/dockerfile:1

#############
# BUILD env #
#############

# Executed in workspace root (NOVELVMP_go)

FROM golang:1.22-bookworm AS builder

# Set destination for COPY
WORKDIR /app

COPY . ./
WORKDIR /app/scanners/scanner_katana

# dependencies for katana
RUN apt-get update && apt-get install -y gcc musl-dev chromium
# && rm -rf /var/lib/apt/lists/*

RUN gcc --version

RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=1 GOOS=linux go build  -ldflags="-w -s" -o /scanner_katana

# ###########
# # RUN env #
# ###########

# FROM gcr.io/distroless/static AS final

RUN useradd -ms /bin/bash nonroot

USER nonroot

# COPY --from=builder /scanner_katana /scanner_katana

ENTRYPOINT ["/scanner_katana"]