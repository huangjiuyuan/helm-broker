package broker

import (
	"fmt"
	"strings"

	osb "github.com/pmorie/go-open-service-broker-client/v2"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
)

// getReleaseStatusCode returns the status code of the release.
func getReleaseStatusCode(status *services.GetReleaseStatusResponse) osb.LastOperationState {
	code := status.GetInfo().GetStatus().GetCode()
	if code == release.Status_DELETING ||
		code == release.Status_PENDING_INSTALL ||
		code == release.Status_PENDING_UPGRADE ||
		code == release.Status_PENDING_ROLLBACK {
		return osb.StateInProgress
	} else if code == release.Status_DEPLOYED ||
		code == release.Status_DELETED {
		return osb.StateSucceeded
	} else {
		return osb.StateFailed
	}
}

// getServiceID returns the service ID from the chart digest.
func getServiceID(id string) (string, error) {
	if len(id) >= 24 {
		id = id[:24]
	} else {
		return "", fmt.Errorf("invalid id pattern")
	}
	return id, nil
}

// getServiceName returns the service name from the chart name.
func getServiceName(name string) (string, error) {
	subStrings := strings.Split(name, "/")
	if len(subStrings) > 1 {
		name = strings.Join(subStrings, ".")
	} else {
		return "", fmt.Errorf("invalid name pattern")
	}
	return name, nil
}

// getChartName returns the chart name from the service ID.
func getChartName(name string) (string, error) {
	if !strings.Contains(name, ".") {
		return "", fmt.Errorf("invalid name pattern")
	}
	r := strings.NewReplacer(".", "/")
	chart := r.Replace(name)
	return chart, nil
}
