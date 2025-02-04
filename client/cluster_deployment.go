package client

import (
	"context"

	"github.com/qovery/qovery-client-go"

	"github.com/qovery/terraform-provider-qovery/client/apierrors"
)

func (c *Client) deployCluster(ctx context.Context, organizationID string, cluster *qovery.Cluster) (*qovery.ClusterStatusGet, *apierrors.APIError) {
	status, apiErr := c.getClusterStatus(ctx, organizationID, cluster.Id)
	if apiErr != nil {
		return nil, apiErr
	}

	switch status.GetStatus() {
	case qovery.STATEENUM_DEPLOYED:
		return status, nil
	default:
		_, res, err := c.api.ClustersApi.
			DeployCluster(ctx, organizationID, cluster.Id).
			Execute()
		if err != nil || res.StatusCode >= 400 {
			return nil, apierrors.NewDeployError(apierrors.APIResourceCluster, cluster.Id, res, err)
		}
	}

	statusChecker := newClusterStatusCheckerWaitFunc(c, organizationID, cluster.Id, qovery.STATEENUM_DEPLOYED)
	if apiErr := wait(ctx, statusChecker, nil); apiErr != nil {
		return nil, apiErr
	}
	return c.getClusterStatus(ctx, organizationID, cluster.Id)
}

func (c *Client) stopCluster(ctx context.Context, organizationID string, cluster *qovery.Cluster) (*qovery.ClusterStatusGet, *apierrors.APIError) {
	status, apiErr := c.getClusterStatus(ctx, organizationID, cluster.Id)
	if apiErr != nil {
		return nil, apiErr
	}

	switch status.GetStatus() {
	case qovery.STATEENUM_STOPPED:
		return status, nil
	default:
		_, res, err := c.api.ClustersApi.
			StopCluster(ctx, organizationID, cluster.Id).
			Execute()
		if err != nil || res.StatusCode >= 400 {
			return nil, apierrors.NewStopError(apierrors.APIResourceCluster, cluster.Id, res, err)
		}
	}

	statusChecker := newClusterStatusCheckerWaitFunc(c, organizationID, cluster.Id, qovery.STATEENUM_STOPPED)
	if apiErr := wait(ctx, statusChecker, nil); apiErr != nil {
		return nil, apiErr
	}
	return c.getClusterStatus(ctx, organizationID, cluster.Id)
}
