# kubectl

KubeCI & Drone plugin to easier use kubectl in your pipeline.

## Build

Build the binary with the following commands:

```
go build -v ./cmd/kubeci-kubectl
```

## Docker

Build the Docker image with the following commands:

```
docker build --rm -t kubeciio/kubectl .
```

## Usage

Execute from the working directory:

```
docker run --rm \
  -e PLUGIN_NAMESPACE=kubeci \
  -e PLUGIN_KUBECTL='get pods' \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  kubeciio/kubectl
```
