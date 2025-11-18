package nist

import (
	pkgnist "github.com/CompassSecurity/pipeleak/pkg/gitlab/nist"
)

func FetchVulns(version string, enterprise bool) (string, error) {
	return pkgnist.FetchVulns(version, enterprise)
}
