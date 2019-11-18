FROM golang:1.12

RUN mkdir /app && \
    chown 10001:10001 /app && \
    groupadd --gid 10001 app && \
    useradd --no-create-home --uid 10001 --gid 10001 --home-dir /app app

ADD . /go/src/go.mozilla.org/repd
RUN mkdir -p /app/bin && \
	GOPATH="/go" go build -o /app/bin/repd go.mozilla.org/repd/cmd/repd

COPY version.json /app/version.json

USER app
WORKDIR /app
CMD /app/bin/repd
