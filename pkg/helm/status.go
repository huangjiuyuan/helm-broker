package helm

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/services"
)

// ReleaseStatus returns the given release's status.
func (c *Client) ReleaseStatus(name string) (*services.GetReleaseStatusResponse, error) {
	resp, err := c.client.ReleaseStatus(
		name,
		helm.StatusReleaseVersion(0),
	)
	if err != nil {
		return nil, prettyError(err)
	}

	return resp, nil
}
