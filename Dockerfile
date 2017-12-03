FROM alpine

RUN apk add --no-cache ca-certificates curl
RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.8.4/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl

ADD ./kubeci-kubectl /usr/bin/

ENTRYPOINT ["/usr/bin/kubeci-kubectl"]
