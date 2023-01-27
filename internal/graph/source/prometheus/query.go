package prometheus

const (
	// 1: type
	// 2: name
	// 3: namespace
	queryFormatSuccessRateSingle = `
	sum by (namespace, %[1]s) (
		irate(
			response_total{classification="success", direction="inbound", %[1]s="%[2]s", namespace="%[3]s"}[5m]
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
	queryFormatEdgesOfUpstreams = `
	sum(
		rate(response_total{%[1]s="%[2]s", namespace="%[3]s"}[5m])
	) by (dst_namespace, dst_deployment, dst_statefulset)
	`

	// 1: type
	// 2: name
	// 3: namespace
	queryFormatEdgesOfDownstreams = `
	sum(
		rate(response_total{dst_%[1]s="%[2]s", dst_namespace="%[3]s"}[5m])
	) by (namespace, deployment, statefulset)
	`
)
