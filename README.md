# stockticker

stockticker is an application that provides an HTTP interface to retrieve a configurable amount of stock data for a specified symbol, and view it in a web browser.

## Assumptions

- Some data is better than nothing so stockticker returns what it can if more data is requested than what is available, indicating the requested amount and the returned amount
- Stale data for potentially multiple hours is acceptable when caching is enabled

# Versions tested with

- Docker: `27.X.Y`
- Helm: `3.16.2`
- Kubernetes: `1.31.0`
- minikube: `1.34.0`

# Building

## Docker

- `$ [sudo] make docker-image`
- Optional: `$ [sudo] make build-in-docker` or `GOOS=darwin make build-in-docker` on macOS
- Optional: `$ [sudo] DOCKER_REPO=<your-docker-repo> DOCKER_IMAGE_TAG=<image-tag> make docker-push`

## Standard

- Ensure at least Go 1.23.2 is installed
- `$ make`

# Usage

## Without caching via Docker

```
$ [sudo] docker run --rm -p 8080:8080 -e SYMBOL=<symbol> -e NDAYS=<days> -e APIKEY=<your-api-key> leonsodhi/stockticker:latest
```

Example:

```
$ [sudo] docker run --rm -p 8080:8080 -e SYMBOL=MSFT -e NDAYS=2 -e APIKEY=key leonsodhi/stockticker:latest
```

## With caching via Docker

Redis is required in this mode.

```
$ [sudo] docker network create st-net
$ [sudo] docker run --name redis --network st-net -d --rm -p 6379:6379 redis:latest
$ [sudo] docker run --network st-net --rm -p 8080:8080 -e SYMBOL=<symbol> -e NDAYS=<days> -e APIKEY=<your-api-key> leonsodhi/stockticker:latest --enable-cache --redis-host redis
$ [sudo] docker network rm st-net
```

## Running natively

```
$ SYMBOL=<symbol> NDAYS=<days> APIKEY=<your-api-key> bin/stockticker
```

## Accessing stock data

Assuming stockticker has been started locally with the default settings, use a web browser to navigate to http://localhost:8080

## All options
```
$ bin/stockticker -h
Usage of stockticker:
      --listen-ip string    The IP address to listen on for HTTP requests (default "0.0.0.0")
      --listen-port int     The port to listen on for HTTP requests (default 8080)
      --enable-cache        Enable/disable caching
      --redis-host string   The Redis host address to connect to (default "127.0.0.1")
      --redis-port int      The Redis port to connect to (default 6379)
```
# Deploying to Kubernetes/minikube

- `minikube start`
- `minikube addons enable ingress`
- `kubectl create namespace stockticker`

## Using plain manifests

For testing purposes, edit the following files and update the associated settings:

- `manifests\configmap.yaml`
  - `data.SYMBOL`
  - `data.NDAYS`
- `manifests\secret.yaml`
  - `data.APIKEY`
    - NOTE: The API key must be base64 encoded using a command such as the following: `echo -n <your-api-key> | base64`

### Start the app without caching

`kubectl apply -f ./manifests`

## Using Helm

For testing purposes, edit `helm/values-local-test.yaml` and update:

- `configMap.data.SYMBOL`
- `configMap.data.NDAYS`
- `secret.data.APIKEY`

Continue to one of the next sections to start the app without caching or with caching.

For production environments:

- Ensure `secret.create` is `false` and remove any key assigned to `APIKEY`
- Create and populate a secret using a solution such as [External Secrets Operator](https://external-secrets.io/latest/) (ESO)
- Specify the name of the secret that ESO (or similar) will create via `envFromSecrets`

### Start the app without caching

- `helm install -f ./helm/values-local-test.yaml -n stockticker stockticker ./helm/stockticker`

### Start the app With caching

- `kubectl create namespace redis`
- `helm install -f ./testing/helm/redis-values.yaml -n redis redis oci://registry-1.docker.io/bitnamicharts/redis`
- `helm install -f ./helm/values-local-test.yaml --set redisCaching.enabled=true --set redisCaching.host=redis-master.redis.svc.cluster.local -n stockticker stockticker ./helm/stockticker`

## Accessing stock data

It may take a minute or so for the pod to become ready and for the ingress to assign an IP. Once both are complete, follow one of the next sections depending on your OS.

### MacOS

- `minikube tunnel`
- Navigate to `localhost` in a browser

### Linux

Navigate to the IP via a web browser. The IP can be retrieved using:

```
kubectl get ingress -n stockticker -o jsonpath='{.items[*].status.loadBalancer.ingress[0].ip}{"\n"}'
```

## Updating values (optional)

- `helm upgrade --reuse-values -f ./helm/values-local-test.yaml -n stockticker stockticker ./helm/stockticker`

# Metrics

Prometheus metrics are exposed on port `9102`

# Profiling

[pprof data](https://pkg.go.dev/net/http/pprof) is available via http://localhost:6060

NOTE: For security reasons, pprof data can only be accessed via localhost, i.e. the pprof server is only listening on this address.

# Development

- `make build` to build the code and generate an executable in the `bin` directory
- `make test` to run tests on the host system
- `[sudo] make test-in-docker` to run the tests in Docker

## Prometheus & Grafana metrics

These steps assume stockticker has been deployed to Kubernetes/minikube.

### Prometheus

Install Prometheus:

```
kubectl create namespace prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update
helm install -f ./testing/helm/prometheus-values.yaml -n prometheus prometheus prometheus-community/prometheus
```

Optionally access Prometheus' web UI via http://localhost:9090 using the following:

```
export POD_NAME=$(kubectl get pods --namespace prometheus -l "app.kubernetes.io/name=prometheus,app.kubernetes.io/instance=prometheus" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace prometheus port-forward $POD_NAME 9090
```

### Grafana

Install Grafana:

```
kubectl create namespace grafana
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install --set adminPassword=password -n grafana grafana grafana/grafana
```

Access Grafana's web UI via http://localhost:3000 with the username `admin` and the password `password`

```
export POD_NAME=$(kubectl get pods --namespace grafana -l "app.kubernetes.io/name=grafana,app.kubernetes.io/instance=grafana" -o jsonpath="{.items[0].metadata.name}")
kubectl --namespace grafana port-forward $POD_NAME 3000
```

Add the Stockticker dashboard:

- Add a new Prometheus data source with `http://prometheus-server.prometheus.svc.cluster.local` as the server URL
- Ensure the scrape interval is set to `15s`
- Import the Grafana dashboard under `./testing`
