FROM golang:1.10.2

RUN groupadd app && useradd -g app -d /app -m app

ADD . /go/src/go.mozilla.org/iprepd
RUN mkdir -p /app/bin && \
	go build -o /app/bin/iprepd go.mozilla.org/iprepd/cmd/iprepd

COPY version.json /app/version.json

USER app
WORKDIR /app
CMD /app/bin/iprepd
