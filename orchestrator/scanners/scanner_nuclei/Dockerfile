# syntax=docker/dockerfile:1

#############
# BUILD env #
#############

# Executed in workspace root (NOVELVMP_go)

FROM golang:1.22 AS builder

# Set destination for COPY
WORKDIR /app

COPY . ./
WORKDIR /app/scanners/scanner_nuclei

RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go build  -ldflags="-w -s" -o /scanner_nuclei

###########
# RUN env #
###########

FROM alpine:latest AS final

RUN adduser nonroot -D

USER nonroot:nonroot

COPY --from=builder /scanner_nuclei /scanner_nuclei

RUN /scanner_nuclei --update

ENTRYPOINT ["/scanner_nuclei"]