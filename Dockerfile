# OFFICIAL REPOSITORY: https://hub.docker.com/_/golang/
FROM --platform=linux/amd64 golang:1.22.2

ENV NAME webdav-serverless
ENV PKG github.com/webdav-serverless/$NAME
ENV SRC_DIR /go/src/$PKG
ENV CMD_DIR $SRC_DIR/cmd/$NAME
RUN mkdir -p $SRC_DIR
WORKDIR $SRC_DIR

# prepare go modules
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . $SRC_DIR

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /go/bin/webdav-serverless ./main.go

FROM gcr.io/distroless/static-debian11
COPY --from=0 /go/bin/webdav-serverless  /go/bin/webdav-serverless
ENTRYPOINT ["/go/bin/webdav-serverless"]