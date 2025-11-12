package nist

import (
	pkgnist "github.com/CompassSecurity/pipeleak/pkg/gitlab/nist"
)

// FetchVulns fetches vulnerabilities for a given GitLab version.
func FetchVulns(version string) (string, error) {
	return pkgnist.FetchVulns(version)
}
