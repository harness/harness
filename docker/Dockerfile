FROM alpine:3 as alpine
RUN apk add -U --no-cache ca-certificates

FROM us.gcr.io/platform-205701/busybox:safe
EXPOSE 80
EXPOSE 443

COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

ENV GODEBUG netdns=go
ADD release/linux/amd64/gitness /bin/
ENTRYPOINT ["/bin/gitness", "server"]
