FROM golang

RUN mkdir -p /go/src/github.com/kujtimiihoxha/kit/test_dir

ADD . /go/src/github.com/kujtimiihoxha/kit/test_dir

RUN go get  -t -v ./...

ENTRYPOINT  go install github.com/kujtimiihoxha/kit/test_dir/strings/cmd && /go/bin/cmd
