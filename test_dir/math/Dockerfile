# Start from a Debian image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

RUN mkdir -p /go/src/github.com/kujtimiihoxha/kit/test_dir

ADD . /go/src/github.com/kujtimiihoxha/kit/test_dir
# Build the outyet command inside the container.
# (You may fetch or manage dependencies here,
# either manually or with a tool like "godep".)
RUN go get  -t -v ./...

ENTRYPOINT  go install github.com/kujtimiihoxha/kit/test_dir/math/cmd && /go/bin/cmd

