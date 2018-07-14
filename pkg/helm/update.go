package helm

import (
	"fmt"
	"strings"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/storage/driver"
)

// UpdateRelease loads a chart from chstr and updates a release to a new/different chart.
func (c *Client) UpdateRelease(chart string, name string) (*services.UpdateReleaseResponse, error) {
	_, err := c.client.ReleaseHistory(name, helm.WithMaxHistory(1))
	if err != nil && strings.Contains(err.Error(), driver.ErrReleaseNotFound(name).Error()) {
		return nil, err
	}

	resp, err := c.client.UpdateRelease(
		name,
		chart,
		helm.UpgradeTimeout(300),
	)
	if err != nil {
		return nil, fmt.Errorf("upgrade failed: %v", prettyError(err))
	}

	return resp, nil
}
