FROM ubuntu:13.10

RUN apt-get update
RUN apt-get install -y wget gcc make g++ build-essential ca-certificates mercurial git bzr

RUN wget https://go.googlecode.com/files/go1.2.src.tar.gz && tar zxvf go1.2.src.tar.gz && \
	cd go/src && ./all.bash 

ENV PATH $PATH:/go/bin:/gocode/bin
ENV GOPATH /gocode

RUN mkdir -p /gocode/src/github.com/drone

ADD . /gocode/src/github.com/drone/drone

WORKDIR /gocode/src/github.com/drone/drone
RUN make deps
RUN make build
EXPOSE 8080
CMD make run
