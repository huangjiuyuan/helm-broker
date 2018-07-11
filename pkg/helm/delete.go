package helm

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/services"
)

// DeleteRelease uninstalls a named release and returns the response.
func (c *Client) DeleteRelease(name string) (*services.UninstallReleaseResponse, error) {
	resp, err := c.client.DeleteRelease(
		name,
		helm.DeletePurge(true),
		helm.DeleteTimeout(300),
	)
	if err != nil {
		return nil, prettyError(err)
	}

	return resp, nil
}
