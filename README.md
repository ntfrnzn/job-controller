### Build

Have a local docker environment

Edit the Makefile to set the version and repository.

```
$ make build push
```

### Deploy

Have a kubernetes cluster accessible with kubectl

Edit the manifest in _resources/jobber-deployment.yaml_ to refer to the correct
version and repository

```
$ resources/create_configmap.sh

$ kubectl create -f resources/jobber-deployment.yaml

```
