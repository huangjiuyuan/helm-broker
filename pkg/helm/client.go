package helm

import (
	"errors"

	"google.golang.org/grpc"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
)

// Client manages client side of helm.
type Client struct {
	// client for helm.
	client helm.Interface
	// settings describes all of the environment settings.
	settings environment.EnvSettings
}

// NewClient creates a new helm client.
func NewClient(host string, home string) *Client {
	cli := &Client{
		client: helm.NewClient(helm.Host(host)),
		settings: environment.EnvSettings{
			TillerHost: host,
			Home:       helmpath.Home(home),
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
