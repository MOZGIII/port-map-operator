FROM golang:1.16.6-buster AS builder

WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -a -o manager -v ./cmd/manager

FROM debian:buster AS pcp-builder
WORKDIR /workspace
RUN apt-get update \
  && apt-get install -y \
  wget \
  tar \
  build-essential \
  cmake
RUN wget -O- https://github.com/libpcp/pcp/tarball/a138a0d34ef8d3f556571d73b8bd6a1008a63d44 | tar -xvz --strip 1
RUN mkdir build \
  && cd build \
  && cmake .. \
  && make
RUN ls -la build

FROM debian:buster
WORKDIR /
COPY --from=builder /workspace/manager /usr/local/bin/manager
COPY --from=pcp-builder /workspace/build/bin/pcp /usr/local/bin/pcp

CMD ["/usr/local/bin/manager"]
