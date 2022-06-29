FROM golang:1.18

ADD . /go/src/go.mozilla.org/iprepd

RUN mkdir -p /app/bin && \
	chown -R 10001:10001 /app /go && \
	groupadd --gid 10001 app && \
	useradd --no-create-home --uid 10001 --gid 10001 --home-dir /app app

USER app

RUN cd /go/src/go.mozilla.org/iprepd && \
	go mod download && \
	go build -o /app/bin/iprepd go.mozilla.org/iprepd/cmd/iprepd

COPY version.json /app/version.json

WORKDIR /app
CMD /app/bin/iprepd
