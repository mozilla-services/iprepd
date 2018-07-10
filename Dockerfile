FROM golang:1.10.2

RUN mkdir /app && \
    chown 10001:10001 /app && \
    groupadd --gid 10001 app && \
    useradd --no-create-home --uid 10001 --gid 10001 --home-dir /app app

ADD . /go/src/go.mozilla.org/iprepd
RUN mkdir -p /app/bin && \
	go build -o /app/bin/iprepd go.mozilla.org/iprepd/cmd/iprepd

COPY version.json /app/version.json

USER app
WORKDIR /app
CMD /app/bin/iprepd
