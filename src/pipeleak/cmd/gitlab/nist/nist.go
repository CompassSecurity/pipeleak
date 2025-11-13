package nist

import (
	pkgnist "github.com/CompassSecurity/pipeleak/pkg/gitlab/nist"
)

func FetchVulns(version string) (string, error) {
	return pkgnist.FetchVulns(version)
}
