package linkerd

import (
	"context"
	"fmt"
	"linkerd-nodegraph/internal/nodegraph"
	"strconv"
	"strings"

	"github.com/prometheus/common/model"
)

var (
	GraphSpec = nodegraph.NodeFields{
		Edge: []nodegraph.Field{
			{Name: "id", Type: nodegraph.FieldTypeString},
			{Name: "source", Type: nodegraph.FieldTypeString},
			{Name: "target", Type: nodegraph.FieldTypeString},
		},
		Node: []nodegraph.Field{
			{Name: "id", Type: nodegraph.FieldTypeString},
			{Name: "title", Type: nodegraph.FieldTypeString},
			{Name: "mainStat", Type: nodegraph.FieldTypeString, DisplayName: "Success Rate"},
			{
				Name:        "arc__failed",
				Type:        nodegraph.FieldTypeNumber,
				Color:       "red",
				DisplayName: "Failed",
			},
			{
				Name:        "arc__success",
				Type:        nodegraph.FieldTypeNumber,
				Color:       "green",
				DisplayName: "Success",
			},
		},
	}

	namespaceLabel     = model.LabelName("namespace")
	dstNamespaceLabel  = model.LabelName("dst_namespace")
	deploymentLabel    = model.LabelName("deployment")
	dstDeploymentLabel = model.LabelName("dst_deployment")
)

type GraphSource interface {
	Query(ctx context.Context, query string) (model.Vector, error)
}

type Stats struct {
	Server GraphSource
}

func (m Stats) successRates(ctx context.Context) (map[model.LabelValue]float64, error) {
	vector, err := m.Server.Query(ctx, "sum(irate(response_total{classification=\"success\", direction=\"inbound\", deployment!=\"\", namespace!=\"\"}[5m])) by (deployment, namespace) / sum(irate(response_total{direction=\"inbound\", deployment!=\"\", namespace!=\"\"}[5m])) by (deployment, namespace) >= 0")
	if err != nil {
		return nil, err
	}

	successRates := map[model.LabelValue]float64{}
	for _, v := range vector {
		successRates[v.Metric[namespaceLabel]+"_"+v.Metric[deploymentLabel]] = float64(v.Value)
	}
	return successRates, nil
}

func (m Stats) Graph(ctx context.Context) (*nodegraph.Graph, error) {
	graph := nodegraph.Graph{Spec: GraphSpec}
	vector, err := m.Server.Query(ctx, "sum(rate(response_total{dst_deployment!=\"\"}[5m])) by (deployment, namespace, dst_namespace, dst_deployment)")
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}

	successRates, err := m.successRates(ctx)
	if err != nil {
		return nil, err
	}

	for _, v := range vector {
		namespace := v.Metric[namespaceLabel]
		deployment := v.Metric[deploymentLabel]
		dstDeployment := v.Metric[dstDeploymentLabel]
		dstNamespace := v.Metric[dstNamespaceLabel]

		graph.AddEdge(nodegraph.Edge{
			"id":     fmt.Sprintf("%s_%s_%s_%s", namespace, deployment, dstNamespace, dstDeployment),
			"source": fmt.Sprintf("%s_%s", namespace, deployment),
			"target": fmt.Sprintf("%s_%s", dstNamespace, dstDeployment),
		})
		seen[fmt.Sprintf("%s_%s", namespace, deployment)] = true
		seen[fmt.Sprintf("%s_%s", dstNamespace, dstDeployment)] = true
	}

	for k := range seen {
		namespace := strings.Split(k, "_")[0]
		deployment := strings.Split(k, "_")[1]
		label := model.LabelValue(k)

		mainStat := strconv.FormatFloat(successRates[label]*100, 'f', 2, 64) + "%"
		if _, ok := successRates[label]; !ok {
			mainStat = "N/A"
		}

		graph.AddNode(nodegraph.Node{
			"id":           k,
			"title":        fmt.Sprintf("%s/%s", namespace, deployment),
			"arc__failed":  1 - successRates[label],
			"arc__success": successRates[label],
			"mainStat":     mainStat,
		})
	}

	return &graph, nil
}
