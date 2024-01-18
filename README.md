# langserve-operator


## Quick start
```bash
# Start minikube cluster
minikube start 

# Deploy Langserve operator
cd operator
make deploy

# Verify operator was deployed (should see a langserve-operator)
kubectl get pods -A

# Create custom Langserve resource
kubectl apply -f config/samples/langsmith.com_v1alpha1_langserve.yaml

# Verify langserve instances deployed (should see 2 langserve pods)
kubectl get pods -A

# Verify langserve CRD exists
kubectl get langserve langserve-sample -o json

# Port-forward (go to localhost:8080) on browser
kubectl port-forward langserve-sample-5c89d699ff-r8jnw 8080:8080

# Delete langserve CRD (pods will be deleted)
kubectl delete langserve langserve-sample

```

## Developer Instructions

To build and push the Langserve docker image (note: this requires Docker repository to already be created)

```bash
cd langserve-template

IMAGE_NAME=njang/langserve
VERSION=0.0.1
docker build . -t $IMAGE_NAME:$VERSION

docker push $IMAGE_NAME:$VERSION
```

To build and push the Operator docker image (note: this requires Docker repository to already be created)

```bash
cd operator

make docker-build docker-push
```

To update the custom [Langserve type](/operator/api/v1alpha1/langserve_types.go), update the go file. Then run `make generate` and `make manifests` to generate the necessary code + manifests.
