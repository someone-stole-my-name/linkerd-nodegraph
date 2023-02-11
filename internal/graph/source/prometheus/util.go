package prometheus

import (
	"context"
	"errors"
	"fmt"
	"linkerd-nodegraph/internal/graph"
	"log"
	"time"

	prom "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

var ErrNotAMatrix = errors.New("expected matrix")

func (prometheus Client) queryRange(ctx context.Context, q string, from int64, to int64) (model.Vector, error) {
	timeRange := prom.Range{
		Start: time.Unix(from/1000, 0),
		End:   time.Unix(to/1000, 0),
		Step:  time.Second * 30,
	}

	res, warn, err := prometheus.API.QueryRange(ctx, q, timeRange)
	if err != nil {
		return nil, fmt.Errorf("query failed: %q: %w", q, err)
	}

	if warn != nil {
		log.Printf("%v", warn)
	}

	if _, ok := res.(model.Matrix); !ok {
		return nil, fmt.Errorf("received '%s': %w", res.Type(), ErrNotAMatrix)
	}

	vector := model.Vector{}

	for _, sampleStream := range res.(model.Matrix) {
		var v float64

		n := len(sampleStream.Values)

		for _, sample := range sampleStream.Values {
			v += float64(sample.Value)
		}

		sample := model.Sample{
			Metric:    sampleStream.Metric,
			Timestamp: sampleStream.Values[0].Timestamp,
			Value:     model.SampleValue(v / float64(n)),
		}

		vector = append(vector, &sample)
	}

	return vector, nil
}

func resourceKindToLabel(k graph.ResourceKind) model.LabelName {
	switch k {
	case graph.DeploymentKind:
		return deploymentLabel
	case graph.StatefulsetKind:
		return statefulsetLabel
	case graph.UndefinedKind:
		fallthrough
	default:
		return deploymentLabel
	}
}

func validEdgeSample(sample model.Sample) bool {
	if _, ok := sample.Metric[dstNamespaceLabel]; !ok {
		return false
	}

	if _, ok := sample.Metric[namespaceLabel]; !ok {
		return false
	}

	return true
}
