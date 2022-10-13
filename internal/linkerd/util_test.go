package linkerd_test

import (
	"context"
	"linkerd-nodegraph/internal/linkerd"
	"linkerd-nodegraph/internal/nodegraph"
	"time"

	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type testCase struct {
	name                    string
	meta                    string
	prometheusEdgesResponse []model.Vector
	prometheusNodesResponse []model.Vector
	edgesExpect             []linkerd.Edge
	nodesExpect             []linkerd.Node
	graphExpect             nodegraph.Graph
}

type outputPromMock struct {
	output []model.Vector
	idx    int
}

func (p *outputPromMock) Query(ctx context.Context, query string, ts time.Time, opts ...prom.Option) (model.Value, prom.Warnings, error) {
	if len(p.output) >= p.idx+1 {
		p.idx++

		return p.output[p.idx-1], nil, nil
	}

	return nil, nil, nil
}

func _float64(x float64) *float64 {
	return &x
}

var (
	testCases = []testCase{emojivoto}

	emojivoto = testCase{
		name: "emojivoto",
		meta: "mocks a mesh with emojivoto app",
		nodesExpect: []linkerd.Node{
			{
				Resource:    linkerd.Resource{Name: "emoji", Namespace: "emojivoto", ResourceType: linkerd.DeploymentResourceType},
				SuccessRate: _float64(1),
			},
			{
				Resource:    linkerd.Resource{Name: "vote-bot", Namespace: "emojivoto", ResourceType: linkerd.DeploymentResourceType},
				SuccessRate: _float64(1),
			},
			{
				Resource:    linkerd.Resource{Name: "voting", Namespace: "emojivoto", ResourceType: linkerd.DeploymentResourceType},
				SuccessRate: _float64(1),
			},
			{
				Resource:    linkerd.Resource{Name: "web", Namespace: "emojivoto", ResourceType: linkerd.DeploymentResourceType},
				SuccessRate: _float64(0.81),
			},
		},
		edgesExpect: []linkerd.Edge{
			{
				Source: linkerd.Resource{
					Name:         "vote-bot",
					Namespace:    "emojivoto",
					ResourceType: linkerd.DeploymentResourceType,
				},
				Destination: linkerd.Resource{
					Name:         "web",
					Namespace:    "emojivoto",
					ResourceType: linkerd.DeploymentResourceType,
				},
			},
			{
				Source: linkerd.Resource{
					Name:         "web",
					Namespace:    "emojivoto",
					ResourceType: linkerd.DeploymentResourceType,
				},
				Destination: linkerd.Resource{
					Name:         "emoji",
					Namespace:    "emojivoto",
					ResourceType: linkerd.DeploymentResourceType,
				},
			},
			{
				Source: linkerd.Resource{
					Name:         "web",
					Namespace:    "emojivoto",
					ResourceType: linkerd.DeploymentResourceType,
				},
				Destination: linkerd.Resource{
					Name:         "voting",
					Namespace:    "emojivoto",
					ResourceType: linkerd.DeploymentResourceType,
				},
			},
		},
		graphExpect: nodegraph.Graph{
			Spec: nodegraph.NodeFields{
				Node: []nodegraph.Field{
					{Name: "id", Type: 0, Color: "", DisplayName: ""},
					{Name: "title", Type: 0, Color: "", DisplayName: "Resource"},
					{Name: "mainStat", Type: 0, Color: "", DisplayName: "Success Rate"},
					{Name: "detail__type", Type: 0, Color: "", DisplayName: "Type"},
					{Name: "detail__namespace", Type: 0, Color: "", DisplayName: "Namespace"},
					{Name: "detail__name", Type: 0, Color: "", DisplayName: "Name"},
					{Name: "arc__failed", Type: 1, Color: "red", DisplayName: "Failed"},
					{Name: "arc__success", Type: 1, Color: "green", DisplayName: "Success"},
				},
				Edge: []nodegraph.Field{
					{Name: "id", Type: 0, Color: "", DisplayName: ""},
					{Name: "source", Type: 0, Color: "", DisplayName: ""},
					{Name: "target", Type: 0, Color: "", DisplayName: ""},
				},
			},
			Nodes: []nodegraph.Node{
				{
					"arc__failed":       float64(0),
					"arc__success":      float64(1),
					"detail__name":      "emoji",
					"detail__namespace": "emojivoto",
					"detail__type":      "deployment",
					"id":                "emojivoto__emoji__deployment",
					"mainStat":          "100.00%", "title": "emojivoto/emoji",
				},
				{
					"arc__failed":       float64(0),
					"arc__success":      float64(1),
					"detail__name":      "vote-bot",
					"detail__namespace": "emojivoto",
					"detail__type":      "deployment",
					"id":                "emojivoto__vote-bot__deployment",
					"mainStat":          "100.00%",
					"title":             "emojivoto/vote-bot",
				},
				{
					"arc__failed":       float64(0),
					"arc__success":      float64(1),
					"detail__name":      "voting",
					"detail__namespace": "emojivoto",
					"detail__type":      "deployment",
					"id":                "emojivoto__voting__deployment",
					"mainStat":          "100.00%",
					"title":             "emojivoto/voting",
				},
				{
					"arc__failed":  0.18999999999999995,
					"arc__success": 0.81,
					"detail__name": "web", "detail__namespace": "emojivoto",
					"detail__type": "deployment",
					"id":           "emojivoto__web__deployment",
					"mainStat":     "81.00%",
					"title":        "emojivoto/web",
				},
			},
			Edges: []nodegraph.Edge{
				{
					"id":     "emojivoto__vote-bot__deployment__emojivoto__web__deployment",
					"source": "emojivoto__vote-bot__deployment",
					"target": "emojivoto__web__deployment",
				}, {
					"id":     "emojivoto__web__deployment__emojivoto__emoji__deployment",
					"source": "emojivoto__web__deployment",
					"target": "emojivoto__emoji__deployment",
				}, {
					"id":     "emojivoto__web__deployment__emojivoto__voting__deployment",
					"source": "emojivoto__web__deployment",
					"target": "emojivoto__voting__deployment",
				},
			},
		},
		prometheusEdgesResponse: []model.Vector{
			{
				&model.Sample{
					Metric: model.Metric{
						"deployment": "emoji",
						"namespace":  "emojivoto",
					},
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment":     "vote-bot",
						"namespace":      "emojivoto",
						"dst_deployment": "web",
						"dst_namespace":  "emojivoto",
					},
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment": "vote-bot",
						"namespace":  "emojivoto",
					},
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment": "voting",
						"namespace":  "emojivoto",
					},
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment":     "web",
						"namespace":      "emojivoto",
						"dst_deployment": "emoji",
						"dst_namespace":  "emojivoto",
					},
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment":     "web",
						"namespace":      "emojivoto",
						"dst_deployment": "voting",
						"dst_namespace":  "emojivoto",
					},
				},
			},
		},
		prometheusNodesResponse: []model.Vector{
			{
				&model.Sample{
					Metric: model.Metric{
						"deployment": "emoji",
						"namespace":  "emojivoto",
					},
					Value:     1,
					Timestamp: 1665602119,
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment": "vote-bot",
						"namespace":  "emojivoto",
					},
					Value:     1,
					Timestamp: 1665602119,
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment": "voting",
						"namespace":  "emojivoto",
					},
					Value:     1,
					Timestamp: 1665602119,
				},
				&model.Sample{
					Metric: model.Metric{
						"deployment": "web",
						"namespace":  "emojivoto",
					},
					Value:     0.81,
					Timestamp: 1665602119,
				},
			},
			// Second query is empty since there are no statefulsets
			{},
		},
	}
)
