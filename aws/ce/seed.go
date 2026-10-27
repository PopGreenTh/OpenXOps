package ce

import (
	"hash/fnv"
	"math"
)

var mockServices = []string{
	"Amazon Elastic Compute Cloud - Compute",
	"Amazon Simple Storage Service",
	"AWS Lambda",
	"AmazonCloudWatch",
}

// mockAmount returns a deterministic positive amount from dimensions.
func mockAmount(account, service, tagValue, period, metric string) float64 {
	h := fnv.New64a()
	for _, s := range []string{account, service, tagValue, period, metric} {
		_, _ = h.Write([]byte(s))
		_, _ = h.Write([]byte{0})
	}
	v := float64(h.Sum64()%9000+100) / 100.0
	return math.Round(v*100) / 100
}
