package format

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

func PrettyPrintYAML(yamlStr string) (string, error) {
	var node yaml.Node

	err := yaml.Unmarshal([]byte(yamlStr), &node)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	err = encoder.Encode(&node)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
