# syntax=docker/dockerfile:1

# http://www.webdav.org/neon/litmus/
# https://github.com/messense/dav-server-rs/blob/main/README.litmus-test.md

FROM arm64v8/gcc:4.9 AS builder
RUN <<EOF
curl -O http://www.webdav.org/neon/litmus/litmus-0.13.tar.gz
tar xf litmus-0.13.tar.gz
cd litmus-0.13
./configure --prefix=/litmus
make install
EOF


FROM arm64v8/ubuntu:20.04
RUN apt update && apt install -y libgssapi-krb5-2 libk5crypto3 libexpat1 && apt-get clean && rm -rf /var/lib/apt/lists/*
WORKDIR /litmus/
COPY --chmod=755 --from=builder /litmus/bin/ /litmus/bin/
COPY --chmod=755 --from=builder /litmus/libexec/litmus/ /litmus/libexec/litmus/
COPY --chmod=755 --from=builder /litmus/share/ /litmus/share/
ENTRYPOINT ["/litmus/bin/litmus"]
