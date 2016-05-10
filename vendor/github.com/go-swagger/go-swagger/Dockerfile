FROM alpine
MAINTAINER go-swagger <ivan+goswagger@flanders.co.nz>

RUN apk --update add ca-certificates shared-mime-info

ADD ./dist/swagger /usr/bin/swagger

ENTRYPOINT ["/usr/bin/swagger"]
