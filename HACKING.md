## Testing against real data locally

1 - Switch to a cluster with prometheus and forward traffic to the service:

```
kontext $CONTEXT
kubectl port-forward svc/prometheus -n $NS 9090:9090
```

2 - Build a local image and run the server:

```
docker build -t ghcr.io/someone-stole-my-name/linkerd-nodegraph:latest .
docker run --rm -p 5001:5001 \
  ghcr.io/someone-stole-my-name/linkerd-nodegraph:latest \
  /nodegraph-server --prometheus-addr http://host.docker.internal:9090
```

3 - Start a new Grafana instance:

```
docker run --rm -p 3000:3000 \
  -e "GF_INSTALL_PLUGINS=hamedkarbasi93-nodegraphapi-datasource" \
  grafana/grafana-enterprise
```

4 - Add a new nodegraph datasource with `http://host.docker.internal:5001` as URL.
