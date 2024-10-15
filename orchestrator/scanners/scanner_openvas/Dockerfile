# syntax=docker/dockerfile:1

#######################
# BUILD env & RUN env #
#######################

# Executed in workspace root (NOVELVMP_go)

FROM python:3.11 AS builder

RUN useradd \
    --no-log-init \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    -r \
    scanner_openvas

# Set destination for COPY
WORKDIR /app

COPY . ./
WORKDIR /app/scanners/scanner_openvas

RUN pip install -r requirements.txt
ENTRYPOINT ["python", "./main.py"]

