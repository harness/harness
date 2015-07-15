# this is an example image.  DO NOT USE THE CERTS IN PRODUCTION
FROM debian:jessie
ADD ./certs /certs
ADD bastion /usr/local/bin/bastion
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/bastion"]
CMD []
