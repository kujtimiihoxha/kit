FROM golang

RUN mkdir -p /go/src/github.com/kujtimiihoxha/kit/test_dir

ADD . /go/src/github.com/kujtimiihoxha/kit/test_dir

RUN go get  -t -v ./...

ENTRYPOINT  go install github.com/kujtimiihoxha/kit/test_dir/math/cmd && /go/bin/cmd
