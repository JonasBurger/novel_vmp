# syntax=docker/dockerfile:1

#############
# BUILD env #
#############

# Executed in workspace root (NOVELVMP_go)

FROM golang:1.22 AS builder

RUN useradd \
    --no-log-init \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    -r \
    scanner_dummy

# Set destination for COPY
WORKDIR /app

COPY . ./
WORKDIR /app/scanners/scanner_dummy

RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /scanner_dummy

###########
# RUN env #
###########

FROM scratch

# Import the user and group files from the builder.
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

USER scanner_dummy:scanner_dummy

COPY --from=builder /scanner_dummy /scanner_dummy

ENTRYPOINT ["/scanner_dummy"]