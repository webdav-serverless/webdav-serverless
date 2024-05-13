# syntax=docker/dockerfile:1

# https://notroj.github.io/litmus/
# https://github.com/messense/dav-server-rs/blob/main/README.litmus-test.md

FROM arm64v8/gcc:4.9 AS builder
RUN <<EOF
curl -O https://notroj.github.io/litmus/litmus-0.14.tar.gz
tar xf litmus-0.14.tar.gz
cd litmus-0.14
./configure --prefix=/litmus
make install
EOF

FROM arm64v8/ubuntu:20.04
RUN apt update && apt install -y libgssapi-krb5-2 libk5crypto3 libexpat1 && apt-get clean && rm -rf /var/lib/apt/lists/*
WORKDIR /litmus/
COPY --chmod=755 --from=builder /litmus/ /litmus/
ENTRYPOINT ["/litmus/bin/litmus"]
