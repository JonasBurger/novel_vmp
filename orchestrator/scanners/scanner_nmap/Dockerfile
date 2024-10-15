# syntax=docker/dockerfile:1

#############
# BUILD env #
#############

# Executed in workspace root (NOVELVMP_go)

FROM golang:1.22 AS builder

# Set destination for COPY
WORKDIR /app

COPY . ./
WORKDIR /app/scanners/scanner_nmap

RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go build  -ldflags="-w -s" -o /scanner_nmap

###########
# RUN env #
###########

FROM instrumentisto/nmap:latest AS final

# run as root

COPY --from=builder /scanner_nmap /scanner_nmap

ENTRYPOINT ["/scanner_nmap"]