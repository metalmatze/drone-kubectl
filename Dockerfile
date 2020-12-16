FROM alpine

RUN apk add --no-cache ca-certificates curl
RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.19.4/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl

ADD ./drone-kubectl /usr/bin/

ENTRYPOINT ["/usr/bin/drone-kubectl"]
