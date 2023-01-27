package prometheus

const (
	// 1: type
	// 2: name
	// 3: namespace
	// 4: additional filter labels
	queryFormatSuccessRateSingle = `
	sum by (namespace, %[1]s) (
		irate(
			response_total{classification="success", direction="inbound", %[1]s="%[2]s", namespace="%[3]s" %[4]s}[5m]
		)
	)  /
	sum by (namespace, %[1]s) (
		irate(
			response_total{direction="inbound", %[1]s!="", namespace!=""}[5m]
		)
	) >= 0`

	// 1: type
	// 2: name
	// 3: namespace
	// 4: additional filter labels
	queryFormatLatencyP95Single = `
	histogram_quantile(
		0.95,
		sum by (le, namespace) (
			rate(response_latency_ms_bucket{%[1]s="%[2]s", namespace="%[3]s", direction="inbound" %[4]s}[5m])
		)
	)
	`

	// 1: type
	// 2: name
	// 3: namespace
	// 4: additional filter labels
	queryFormatRequestVolumeSingle = `
	sum by (namespace) (
		rate(request_total{%[1]s="%[2]s", namespace="%[3]s", direction="inbound" %[4]s}[5m])
	) 
	`

	// 1: type
	// 2: name
	// 3: namespace
	// 4: additional filter labels
	queryFormatEdgesOfUpstreams = `
	sum(
		response_total{%[1]s="%[2]s", namespace="%[3]s" %[4]s}
	) by (dst_namespace, dst_deployment, dst_statefulset)
	`

	// 1: type
	// 2: name
	// 3: namespace
	// 4: additional filter labels
	queryFormatEdgesOfDownstreams = `
	sum(
		response_total{dst_%[1]s="%[2]s", dst_namespace="%[3]s" %[4]s}
	) by (namespace, deployment, statefulset)
	`
)
