FROM golang:1.24 as builder

WORKDIR /app

COPY . .

RUN make build

#####################################################################

FROM ubuntu:24.04

RUN useradd appuser

WORKDIR /app

COPY --from=builder /app/rollups-avail-reader .

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080 5004

ENTRYPOINT ["/app/rollups-avail-reader"]