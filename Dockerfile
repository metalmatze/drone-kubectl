FROM alpine

RUN apk add --no-cache ca-certificates curl
RUN curl -Lo /usr/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/v1.12.1/bin/linux/amd64/kubectl && \
    chmod +x /usr/bin/kubectl

ADD ./kubeciio-kubectl /usr/bin/

ENTRYPOINT ["/usr/bin/kubeciio-kubectl"]
