FROM ubuntu
ENV  DRONE_SERVER_PORT :80

ADD packaging/output/ /dronepkg
WORKDIR /dronepkg

RUN apt-get update
RUN apt-get -y install zip libsqlite3-dev sqlite3 1> /dev/null 2> /dev/null
RUN make deps build embed install

EXPOSE 80
ENTRYPOINT ["/usr/local/bin/droned"]
