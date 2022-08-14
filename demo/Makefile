REQ=linkerd kind helm kubectl curl

all: clean kubernetes

clean: kind_destroy

build:
	docker build -t linkerd-grafana-graph:latest ../
	kind load docker-image \
		--name linkerd-grafana-graph \
		linkerd-grafana-graph:latest

kubernetes_requirements:
	$(foreach bin,$(REQ),\
    $(if $(shell command -v $(bin) 2> /dev/null),$(info found `$(bin)`),$(error please install `$(bin)`)))

kind_create:
	kind create cluster \
		--config manifests/cluster.yaml \
		--name linkerd-grafana-graph

kind_destroy:
	kind delete cluster \
		--name linkerd-grafana-graph

helm_repo:
	helm repo add \
		prometheus-community \
		https://prometheus-community.github.io/helm-charts
	helm repo add \
		grafana \
		https://grafana.github.io/helm-charts
	helm repo update

linkerd:
	linkerd check --pre
	linkerd install | kubectl apply -f -
	linkerd check
	linkerd viz install \
		--set prometheusUrl=http://prometheus.default \
		--set prometheus.enabled=false \
		--set grafana.enabled=false |\
		kubectl apply -f -
	kubectl apply -f manifests/linkerd-web-svc.yaml
	curl -sSfL \
		https://run.linkerd.io/emojivoto.yml |\
		kubectl apply -f -

helm_install: helm_repo
	helm install \
		prometheus \
		prometheus-community/prometheus \
		-f helm/prometheus.yaml 
	kubectl apply -f manifests/prometheus-svc.yaml
	helm install \
		grafana \
		grafana/grafana \
		-f helm/grafana.yaml

kubernetes: kubernetes_requirements clean kind_create linkerd helm_install build
	kubectl get -n emojivoto deploy -o yaml |\
		linkerd inject - |\
		kubectl apply -f -
	kubectl apply -f manifests/linkerd-grafana-graph.yaml
