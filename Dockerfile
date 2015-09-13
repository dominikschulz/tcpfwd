FROM golang:1.5

ENV GOBIN /go/bin
ENV GOPATH /go
ADD . /go/src/github.com/dominikschulz/tcpfwd
WORKDIR /go/src/github.com/dominikschulz/tcpfwd
RUN go get ./...
RUN go install

CMD [ "/go/bin/tcpfwd" ]

EXPOSE 8080

