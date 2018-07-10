package broker

import (
	"fmt"
	"net/http"
	"reflect"
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
		async:     o.Async,
		instances: make(map[string]*exampleInstance, 10),
	}, nil
}

// HelmBroker provides an implementation of the broker.Interface.
type HelmBroker struct {
	// Indicates if the broker should handle the requests asynchronously.
	async bool
	// Synchronize go routines.
	sync.RWMutex
	// Add fields here! These fields are provided purely as an example
	instances map[string]*exampleInstance
}

var _ broker.Interface = &HelmBroker{}

// GetCatalog encapsulates the business logic for returning the broker's catalog of services.
func (b *HelmBroker) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	hc := helm.NewClient(":44134", "$HOME/.helm")
	releases, err := hc.SearchReleases()
	if err != nil {
		glog.Errorf("failed to get releases from Chart repositories")
	}

	response := &broker.CatalogResponse{}
	services := make([]osb.Service, len(releases), len(releases))
	for idx, release := range releases {
		service := osb.Service{
			Name:        release.Name,
			ID:          release.Chart.Digest,
			Description: release.Chart.Description,
			Bindable:    true,
			Metadata: map[string]interface{}{
				"name":          release.Chart.Name,
				"home":          release.Chart.Home,
				"sources":       release.Chart.Sources,
				"version":       release.Chart.Version,
				"description":   release.Chart.Description,
				"keywords":      release.Chart.Keywords,
				"maintainers":   release.Chart.Maintainers,
				"engine":        release.Chart.Engine,
				"apiVersion":    release.Chart.ApiVersion,
				"condition":     release.Chart.Condition,
				"tags":          release.Chart.Tags,
				"appVersion":    release.Chart.AppVersion,
				"deprecated":    release.Chart.Deprecated,
				"tillerVersion": release.Chart.TillerVersion,
				"annotations":   release.Chart.Annotations,
			},
			Plans: []osb.Plan{
				{
					Name:        "default",
					ID:          "0",
					Description: fmt.Sprintf("The default plan for %s service.", release.Chart.Name),
					Free:        func() *bool { b := true; return &b }(),
					Schemas: &osb.Schemas{
						ServiceInstance: &osb.ServiceInstanceSchema{
							Create: &osb.InputParametersSchema{
								Parameters: map[string]string{
									"name":    release.Name,
									"version": release.Chart.Version,
								},
							},
						},
					},
				},
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

func (b *HelmBroker) Provision(request *osb.ProvisionRequest, c *broker.RequestContext) (*broker.ProvisionResponse, error) {
	// Your provision business logic goes here

	// example implementation:
	b.Lock()
	defer b.Unlock()

	response := broker.ProvisionResponse{}

	exampleInstance := &exampleInstance{
		ID:        request.InstanceID,
		ServiceID: request.ServiceID,
		PlanID:    request.PlanID,
		Params:    request.Parameters,
	}

	// Check to see if this is the same instance
	if i := b.instances[request.InstanceID]; i != nil {
		if i.Match(exampleInstance) {
			response.Exists = true
			return &response, nil
		} else {
			// Instance ID in use, this is a conflict.
			description := "InstanceID in use"
			return nil, osb.HTTPStatusCodeError{
				StatusCode:  http.StatusConflict,
				Description: &description,
			}
		}
	}
	b.instances[request.InstanceID] = exampleInstance

	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *HelmBroker) Deprovision(request *osb.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	// Your deprovision business logic goes here

	// example implementation:
	b.Lock()
	defer b.Unlock()

	response := broker.DeprovisionResponse{}

	delete(b.instances, request.InstanceID)

	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *HelmBroker) LastOperation(request *osb.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	// Your last-operation business logic goes here

	return nil, nil
}

func (b *HelmBroker) Bind(request *osb.BindRequest, c *broker.RequestContext) (*broker.BindResponse, error) {
	// Your bind business logic goes here

	// example implementation:
	b.Lock()
	defer b.Unlock()

	instance, ok := b.instances[request.InstanceID]
	if !ok {
		return nil, osb.HTTPStatusCodeError{
			StatusCode: http.StatusNotFound,
		}
	}

	response := broker.BindResponse{
		BindResponse: osb.BindResponse{
			Credentials: instance.Params,
		},
	}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *HelmBroker) Unbind(request *osb.UnbindRequest, c *broker.RequestContext) (*broker.UnbindResponse, error) {
	// Your unbind business logic goes here
	return &broker.UnbindResponse{}, nil
}

func (b *HelmBroker) Update(request *osb.UpdateInstanceRequest, c *broker.RequestContext) (*broker.UpdateInstanceResponse, error) {
	// Your logic for updating a service goes here.
	response := broker.UpdateInstanceResponse{}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	return &response, nil
}

func (b *HelmBroker) ValidateBrokerAPIVersion(version string) error {
	return nil
}

// example types

// exampleInstance is intended as an example of a type that holds information about a service instance
type exampleInstance struct {
	ID        string
	ServiceID string
	PlanID    string
	Params    map[string]interface{}
}

func (i *exampleInstance) Match(other *exampleInstance) bool {
	return reflect.DeepEqual(i, other)
}
