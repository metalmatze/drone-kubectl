---
date: 2018-10-15T00:00:00+00:00
title: drone-kubectl
author: metalmatze
tags: [ kubernetes, kubectl ]
repo: metalmatze/drone-kubectl
logo: kubectl.svg
image: metalmatze/drone-kubectl
---

The kubectl plugin can be used to do everything kubectl can do.
It provides some helpers and shortcuts to make your pipeline step more readable.

Example configuration running inside a kubernetes cluster:

```yaml
pipeline:
  kubectl:
    image: metalmatze/drone-kubectl
    kubectl: get pods
```

Example configuration using a kubeconfig via a secret:

This is **required** when running outside a cluster or a different one should be talked to. 

```diff
pipeline:
  kubectl:
    image: metalmatze/drone-kubectl
    kubectl: get pods
+   secrets: [ kubeconfig ]
```

_This takes the secret kubeconfig, base64 decodes it and writes it to disk ready for kubectl._ 

Example configuration using a different kubeconig via a secrets:

```diff
pipeline:
  kubectl:
    image: metalmatze/drone-kubectl
    kubectl: get pods
+   secrets:
+     - source: kubeconfig_development
+       target: kubeconfig
```

_This maps the `kubeconfig_develpment` secret to be used by the plugin as `kubeconfig` which is then forwarded to kubectl._

Example configuration running inside a kubernetes cluster using a different namespace:

```diff
pipeline:
  kubectl:
    image: metalmatze/drone-kubectl
-   kubectl: get pods
+   kubectl: --namespace application get pods
```

equivalent:

```diff
pipeline:
  kubectl:
    image: metalmatze/drone-kubectl
    kubectl: get pods
+   namespace: application
```

Example configuration using file paths to apply:

```diff
pipeline:
  kubectl:
    image: metalmatze/drone-kubectl
+   kubectl: apply -f /path/to/folder/foo.yaml -f /path/to/folder/bar.yaml -f /path/to/folder/baz.yaml
```

equivalent but easier to read:

```diff
pipeline:
  kubectl:
    image: metalmatze/drone-kubectl
-   kubectl: apply -f /path/to/folder/foo.yaml -f /path/to/folder/bar.yaml -f /path/to/folder/baz.yaml
+   kubectl: apply
+   files: 
+     - /path/to/folder/foo.yaml 
+     - /path/to/folder/bar.yaml 
+     - /path/to/folder/baz.yaml
```

### Templating

Templating generate commands or files to be used by kubectl when executing.

Setting the image to the current commit:

```diff
pipeline:
  kubectl:
   image: metalmatze/drone-kubectl
   kubectl: set image deployment/foo container=bar/baz:{{ .DroneCommit }}
```

### Debug

You can turn on the debug mode to see some details like the plain text `kubeconfig`. **Attention** only use this on test systems, **NOT AT PRODUCTIVE SYSTEMS**.

```diff
pipeline:
  kubectl:
   image: metalmatze/drone-kubectl
+  debug: true  
```
