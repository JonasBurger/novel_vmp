# syntax=docker/dockerfile:1

#############
# BUILD env #
#############

# Executed in workspace root (NOVELVMP_go)

FROM golang:1.22 AS builder

# Set destination for COPY
WORKDIR /app

COPY . ./
WORKDIR /app/scanners/scanner_subfinder

RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go build  -ldflags="-w -s" -o /scanner_subfinder

###########
# RUN env #
###########

FROM gcr.io/distroless/static AS final

USER nonroot:nonroot

COPY --from=builder /scanner_subfinder /scanner_subfinder

ENTRYPOINT ["/scanner_subfinder"]