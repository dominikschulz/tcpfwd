FROM golang:1.8-alpine3.6 as builder

ADD . /go/src/github.com/dominikschulz/tcpfwd
WORKDIR /go/src/github.com/dominikschulz/tcpfwd

RUN go install

FROM alpine:3.6

COPY --from=builder /go/bin/tcpfwd /usr/local/bin/tcpfwd
CMD [ "/usr/local/bin/tcpfwd" ]
EXPOSE 8080
