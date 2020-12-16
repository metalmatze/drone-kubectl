# drone-kubectl

Drone plugin for easier use of kubectl in your pipelines.

_This work is based on the previous KubeCI/kubectl plugin._

## Build

Build the binary with the following commands:

```
go build -v ./cmd/drone-kubectl
```

## Docker

Build the Docker image with the following commands:

```
docker build --rm -t metalmatze/drone-kubectl .
```

## Usage

Execute from the working directory:

```
docker run --rm \
  -e PLUGIN_NAMESPACE=drone \
  -e PLUGIN_KUBECTL='get pods' \
  -v $(pwd):$(pwd) \
  -w $(pwd) \
  metalmatze/drone-kubectl
```
