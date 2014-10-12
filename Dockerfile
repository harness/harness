# This is a Docker image for the Drone CI system.
# Use the following command to start the container:
#    docker run -p 127.0.0.1:80:80 -t drone/drone

FROM google/golang

RUN apt-get update
RUN apt-get -y install zip libsqlite3-dev sqlite3 1> /dev/null 2> /dev/null

ADD . /gopath/src/github.com/drone/drone/
WORKDIR /gopath/src/github.com/drone/drone

RUN make deps build embed test install 

EXPOSE 80

ENTRYPOINT ["/usr/local/bin/droned"]
CMD ["--bind=:80"]
