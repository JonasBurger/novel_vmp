# syntax=docker/dockerfile:1

#############
# BUILD env #
#############

# Executed in workspace root (NOVELVMP_go)

FROM golang:1.22 AS builder

# Set destination for COPY
WORKDIR /app

COPY . ./
WORKDIR /app/scanners/scanner_wapalyzer

RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 GOOS=linux go build  -ldflags="-w -s" -o /scanner_wapalyzer

###########
# RUN env #
###########

FROM gcr.io/distroless/static AS final

USER nonroot:nonroot

COPY --from=builder /app/scanners/scanner_wapalyzer/wapalyzer_data /wapalyzer_data
COPY --from=builder /scanner_wapalyzer /scanner_wapalyzer

ENTRYPOINT ["/scanner_wapalyzer"]