package helm

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/services"
)

// InstallRelease loads a chart, installs it, and returns the release response.
func (c *Client) InstallRelease(chart string, namespace string, name string) (*services.InstallReleaseResponse, error) {
	resp, err := c.client.InstallRelease(
		chart,
		namespace,
		helm.ReleaseName(name),
		helm.InstallTimeout(300),
	)
	if err != nil {
		return nil, prettyError(err)
	}

	return resp, nil
}
