package broker

import (
	"sync"

	"github.com/golang/glog"
	"github.com/huangjiuyuan/helm-broker/pkg/helm"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/osb-broker-lib/pkg/broker"
)

// NewHelmBroker is a hook that is called with the Options the program is run
// with. NewHelmBroker is the place where you will initialize your
// HelmBroker the parameters passed in.
func NewHelmBroker(o Options) (*HelmBroker, error) {
	// For example, if your HelmBroker requires a parameter from the command
	// line, you would unpack it from the Options and set it on the
	// HelmBroker here.
	return &HelmBroker{
		async:   o.Async,
		client:  helm.NewClient("192.168.99.100:30400", "$HOME/.helm"),
		version: "2.13",
	}, nil
}

// HelmBroker provides an implementation of the broker.Interface.
type HelmBroker struct {
	// Indicates if the broker should handle the requests asynchronously.
	async bool
	// Synchronize go routines.
	sync.RWMutex
	// Helm client.
	client *helm.Client
	// API version for broker.
	version string
}

var _ broker.Interface = &HelmBroker{}

// GetCatalog encapsulates the business logic for returning the broker's catalog of services.
func (b *HelmBroker) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	releases, err := b.client.SearchReleases()
	if err != nil {
		glog.Errorf("failed to get releases from Chart repositories")
	}

	response := &broker.CatalogResponse{}
	services := make([]osb.Service, len(releases), len(releases))
	for idx, release := range releases {
		service := osb.Service{
			Name:        release.Chart.Name,
			ID:          release.Name,
			Description: release.Chart.Description,
			Bindable:    false,
			Metadata: map[string]interface{}{
				"name":          release.Chart.Name,
				"home":          release.Chart.Home,
				"sources":       release.Chart.Sources,
				"version":       release.Chart.Version,
				"description":   release.Chart.Description,
				"keywords":      release.Chart.Keywords,
				"maintainers":   release.Chart.Maintainers,
				"engine":        release.Chart.Engine,
				"icon":          release.Chart.Icon,
				"apiVersion":    release.Chart.ApiVersion,
				"condition":     release.Chart.Condition,
				"tags":          release.Chart.Tags,
				"appVersion":    release.Chart.AppVersion,
				"deprecated":    release.Chart.Deprecated,
				"tillerVersion": release.Chart.TillerVersion,
				"annotations":   release.Chart.Annotations,
				"kubeVersion":   release.Chart.KubeVersion,
				"urls":          release.Chart.URLs,
				"created":       release.Chart.Created,
				"removed":       release.Chart.Removed,
				"digest":        release.Chart.Digest,
			},
		}
		services[idx] = service
	}
	osbResponse := &osb.CatalogResponse{
		Services: services,
	}

	glog.Infof("catalog response: %#+v.", osbResponse)
	response.CatalogResponse = *osbResponse

	return response, nil
}

// Provision encapsulates the business logic for a provision operation and returns a osb.ProvisionResponse or an error.
func (b *HelmBroker) Provision(request *osb.ProvisionRequest, c *broker.RequestContext) (*broker.ProvisionResponse, error) {
	resp, err := b.client.InstallRelease(request.ServiceID, "", "", request.Parameters)
	if err != nil {
		return nil, err
	}

	response := broker.ProvisionResponse{
		ProvisionResponse: osb.ProvisionResponse{
			DashboardURL: func() *string { s := ""; return &s }(),
			OperationKey: nil,
		},
	}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}
	release := resp.GetRelease()
	glog.Infof("provision response: %#+v.", response)
	glog.Infof("release %s installed from chart %s.", release.Name, release.Chart.Metadata.Name)

	return &response, nil
}

// Deprovision encapsulates the business logic for a deprovision operation and returns a osb.DeprovisionResponse or an error.
func (b *HelmBroker) Deprovision(request *osb.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	resp, err := b.client.DeleteRelease(request.InstanceID)
	if err != nil {
		return nil, err
	}

	response := broker.DeprovisionResponse{
		DeprovisionResponse: osb.DeprovisionResponse{
			OperationKey: nil,
		},
	}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}
	release := resp.GetRelease()
	glog.Infof("deprovision response: %#+v.", response)
	glog.Infof("release %s from chart %s uninstalled", release.Name, release.Chart.Metadata.Name)

	return &response, nil
}

// LastOperation encapsulates the business logic for a last operation request and returns a osb.LastOperationResponse or an error.
func (b *HelmBroker) LastOperation(request *osb.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	resp, err := b.client.ReleaseStatus(request.InstanceID)
	if err != nil {
		return nil, err
	}

	state := getReleaseStatusCode(resp)
	response := broker.LastOperationResponse{
		LastOperationResponse: osb.LastOperationResponse{
			State: state,
		},
	}

	return &response, nil
}

// Bind encapsulates the business logic for a bind operation and returns a osb.BindResponse or an error.
func (b *HelmBroker) Bind(request *osb.BindRequest, c *broker.RequestContext) (*broker.BindResponse, error) {
	response := broker.BindResponse{
		BindResponse: osb.BindResponse{
			Credentials: nil,
		},
	}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

// Unbind encapsulates the business logic for an unbind operation and returns a osb.UnbindResponse or an error.
func (b *HelmBroker) Unbind(request *osb.UnbindRequest, c *broker.RequestContext) (*broker.UnbindResponse, error) {
	return &broker.UnbindResponse{}, nil
}

// Update encapsulates the business logic for an update operation and returns a osb.UpdateInstanceResponse or an error.
func (b *HelmBroker) Update(request *osb.UpdateInstanceRequest, c *broker.RequestContext) (*broker.UpdateInstanceResponse, error) {
	resp, err := b.client.UpdateRelease(request.InstanceID, request.ServiceID)
	if err != nil {
		return nil, err
	}

	response := broker.UpdateInstanceResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}
	release := resp.GetRelease()
	glog.Infof("update response: %#+v.", response)
	glog.Infof("release %s from chart %s updated", release.Name, release.Chart.Metadata.Name)

	return &response, nil
}

// ValidateBrokerAPIVersion encapsulates the business logic of validating the OSB API version sent to the broker with every request and returns an error.
func (b *HelmBroker) ValidateBrokerAPIVersion(version string) error {
	return nil
}
