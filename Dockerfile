FROM golang:1.15-buster AS builder

WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -a -o manager -v ./cmd/manager

FROM debian:buster
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
