package helm

import (
	"errors"

	"google.golang.org/grpc"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
)

// Client manages client side of Helm.
type Client struct {
	// home specifies the path of Helm.
	home helmpath.Home
	// client for Helm.
	client helm.Interface
	// settings describes all of the environment settings.
	settings environment.EnvSettings
}

// NewClient creates a new Helm client.
func NewClient(host string, home helmpath.Home) *Client {
	cli := &Client{
		home:   home,
		client: helm.NewClient(helm.Host(host)),
		settings: environment.EnvSettings{
			TillerHost: host,
			Home:       home,
		},
	}

	return cli
}

// prettyError unwraps or rewrites certain errors to make them more user-friendly.
func prettyError(err error) error {
	if err == nil {
		return nil
	}

	return errors.New(grpc.ErrorDesc(err))
}
