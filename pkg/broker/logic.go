package broker

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/golang/glog"
	"github.com/huangjiuyuan/helm-broker/pkg/helm"
	svcatclientset "github.com/kubernetes-incubator/service-catalog/pkg/client/clientset_generated/clientset"
	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"github.com/pmorie/osb-broker-lib/pkg/broker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// NewHelmBroker is a hook that is called with the Options the program is run
// with. NewHelmBroker is the place where you will initialize your
// HelmBroker the parameters passed in.
func NewHelmBroker(o Options) (*HelmBroker, error) {
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config when creating helm broker: %v", err)
	}

	// Create the kubernetes clientset.
	kubeClient, err := kubeclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	// Create the service catalog clientset.
	svcatClient, err := svcatclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create service catalog client: %v", err)
	}

	return &HelmBroker{
		async:       o.Async,
		kubeClient:  kubeClient,
		svcatClient: svcatClient,
		helmClient:  helm.NewClient(o.TillerHost, o.HelmHome),
		version:     "2.13",
	}, nil
}

// HelmBroker provides an implementation of the broker.Interface.
type HelmBroker struct {
	// Indicates if the broker should handle the requests asynchronously.
	async bool
	// Clientset for kubernetes.
	kubeClient kubeclientset.Interface
	// Clientset for service catalog.
	svcatClient svcatclientset.Interface
	// Client for helm.
	helmClient *helm.Client
	// API version for broker.
	version string
}

var _ broker.Interface = &HelmBroker{}

// GetCatalog encapsulates the business logic for returning the broker's catalog of services.
func (b *HelmBroker) GetCatalog(c *broker.RequestContext) (*broker.CatalogResponse, error) {
	releases, err := b.helmClient.SearchReleases()
	if err != nil {
		return nil, fmt.Errorf("failed to get releases from Chart repositories")
	}

	response := &broker.CatalogResponse{}
	services := make([]osb.Service, len(releases), len(releases))
	for i, release := range releases {
		serviceID, err := getServiceID(release.Chart.Digest)
		if err != nil {
			glog.Errorf("failed to get service ID for release %s: %v", release.Name, err)
			continue
		}

		serviceName, err := getServiceName(release.Name)
		if err != nil {
			glog.Errorf("failed to get service name for release %s: %v", release.Name, err)
			continue
		}

		service := osb.Service{
			Name:        serviceName,
			ID:          serviceID,
			Description: release.Chart.Description,
			Bindable:    false,
			Plans: []osb.Plan{
				{
					ID:          serviceID,
					Name:        serviceName,
					Description: fmt.Sprintf("A default plan for %s", serviceName),
					Free:        func() *bool { b := true; return &b }(),
				},
			},
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
		services[i] = service
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
	// Get service class for provision request.
	class, err := b.svcatClient.ServicecatalogV1beta1().ClusterServiceClasses().Get(request.ServiceID, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	chart, err := getChartName(class.Spec.ExternalName)
	if err != nil {
		return nil, fmt.Errorf("failed to get chart name for service %s: %v", request.ServiceID, err)
	}

	namespace, ok := request.Context["namespace"].(string)
	if !ok {
		return nil, fmt.Errorf("failed to get namespace for instance %s", request.InstanceID)
	}

	// Get instance for provision request.
	instanceList, err := b.svcatClient.ServicecatalogV1beta1().ServiceInstances(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var name string
	for _, instance := range instanceList.Items {
		if instance.Spec.ExternalID == request.InstanceID {
			name = instance.Name
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("failed to get name for instance %s", request.InstanceID)
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

	// Install helm release.
	resp, err := b.helmClient.InstallRelease(chart, namespace, name, request.Parameters)
	if err != nil {
		return nil, err
	}

	release := resp.GetRelease()
	glog.Infof("provision response: %#+v.", response)
	glog.Infof("release %s installed from chart %s.", release.Name, release.Chart.Metadata.Name)

	return &response, nil
}

// Deprovision encapsulates the business logic for a deprovision operation and returns a osb.DeprovisionResponse or an error.
func (b *HelmBroker) Deprovision(request *osb.DeprovisionRequest, c *broker.RequestContext) (*broker.DeprovisionResponse, error) {
	// Get instance for provision request.
	instanceList, err := b.svcatClient.ServicecatalogV1beta1().ServiceInstances("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var name string
	for _, instance := range instanceList.Items {
		if instance.Spec.ExternalID == request.InstanceID {
			name = instance.Name
			break
		}
	}
	if name == "" {
		return nil, fmt.Errorf("failed to get name for instance %s", request.InstanceID)
	}

	response := broker.DeprovisionResponse{
		DeprovisionResponse: osb.DeprovisionResponse{
			OperationKey: nil,
		},
	}
	if request.AcceptsIncomplete {
		response.Async = b.async
	}

	resp, err := b.helmClient.DeleteRelease(name)
	if err != nil {
		if isReleaseNotFoundError(name, err) {
			return &response, nil
		}
		return nil, err
	}

	release := resp.GetRelease()
	glog.Infof("deprovision response: %#+v.", response)
	glog.Infof("release %s from chart %s uninstalled", release.Name, release.Chart.Metadata.Name)

	return &response, nil
}

// LastOperation encapsulates the business logic for a last operation request and returns a osb.LastOperationResponse or an error.
func (b *HelmBroker) LastOperation(request *osb.LastOperationRequest, c *broker.RequestContext) (*broker.LastOperationResponse, error) {
	resp, err := b.helmClient.ReleaseStatus(request.InstanceID)
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
	resp, err := b.helmClient.UpdateRelease(request.InstanceID, request.ServiceID)
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
