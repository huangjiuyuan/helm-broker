package helm

import (
	"k8s.io/helm/pkg/helm/helmpath"
)

// Client manages client side of Helm.
type Client struct {
	// Host specifies the host address of the Tiller release server.
	Host string
	// HelmHome specifies the path of Helm.
	HelmHome helmpath.Home
}

// NewClient creates a new Helm client.
func NewClient(host string, home helmpath.Home) *Client {
	return &Client{
		Host:     host,
		HelmHome: home,
	}
}
