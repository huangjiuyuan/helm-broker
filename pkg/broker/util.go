package broker

import (
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
