package client

import (
	"context"

	"github.com/qovery/qovery-client-go"

	"github.com/qovery/terraform-provider-qovery/client/apierrors"
)

type ClusterResponse struct {
	OrganizationID         string
	ClusterResponse        *qovery.Cluster
	ClusterInfo            *qovery.ClusterCloudProviderInfo
	ClusterRoutingTable    *ClusterRoutingTable
	ClusterAdvancedSetting *map[string]interface{}
}

type ClusterUpsertParams struct {
	ClusterRequest              qovery.ClusterRequest
	ClusterCloudProviderRequest *qovery.ClusterCloudProviderInfoRequest
	ClusterRoutingTable         ClusterRoutingTable
	ClusterAdvancedSettings     map[string]interface{}
	ForceUpdate                 bool
	DesiredState                qovery.StateEnum
}

func (c *Client) CreateCluster(ctx context.Context, organizationID string, params *ClusterUpsertParams) (*ClusterResponse, *apierrors.APIError) {
	cluster, res, err := c.api.ClustersApi.
		CreateCluster(ctx, organizationID).
		ClusterRequest(params.ClusterRequest).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		return nil, apierrors.NewCreateError(apierrors.APIResourceCluster, params.ClusterRequest.Name, res, err)
	}
	return c.updateCluster(ctx, organizationID, cluster, params)
}

func (c *Client) GetCluster(ctx context.Context, organizationID string, clusterID string) (*ClusterResponse, *apierrors.APIError) {
	cluster, apiErr := c.getClusterByID(ctx, organizationID, clusterID)
	if apiErr != nil {
		return nil, apiErr
	}

	clusterInfo, res, err := c.api.ClustersApi.
		GetOrganizationCloudProviderInfo(ctx, organizationID, cluster.Id).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		return nil, apierrors.NewCreateError(apierrors.APIResourceClusterCloudProvider, cluster.Id, res, err)
	}

	clusterRoutingTable, apiErr := c.getClusterRoutingTable(ctx, organizationID, clusterID)
	if apiErr != nil {
		return nil, apiErr
	}

	clusterSettings, apiErr := c.getClusterAdvancedSettings(ctx, organizationID, clusterID)
	if apiErr != nil {
		return nil, apiErr
	}

	return &ClusterResponse{
		OrganizationID:         organizationID,
		ClusterResponse:        cluster,
		ClusterRoutingTable:    clusterRoutingTable,
		ClusterInfo:            clusterInfo,
		ClusterAdvancedSetting: clusterSettings,
	}, nil
}

func (c *Client) UpdateCluster(ctx context.Context, organizationID string, clusterID string, params *ClusterUpsertParams) (*ClusterResponse, *apierrors.APIError) {
	cluster, res, err := c.api.ClustersApi.
		EditCluster(ctx, organizationID, clusterID).
		ClusterRequest(params.ClusterRequest).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		return nil, apierrors.NewUpdateError(apierrors.APIResourceCluster, clusterID, res, err)
	}

	return c.updateCluster(ctx, organizationID, cluster, params)
}

func (c *Client) DeleteCluster(ctx context.Context, organizationID string, clusterID string) *apierrors.APIError {
	finalStateChecker := newClusterFinalStateCheckerWaitFunc(c, organizationID, clusterID)
	if apiErr := wait(ctx, finalStateChecker, nil); apiErr != nil {
		return apiErr
	}

	res, err := c.api.ClustersApi.
		DeleteCluster(ctx, organizationID, clusterID).
		Execute()
	if err != nil || res.StatusCode >= 300 {
		return apierrors.NewDeleteError(apierrors.APIResourceCluster, clusterID, res, err)
	}

	checker := newClusterStatusCheckerWaitFunc(c, organizationID, clusterID, "DELETED")
	if apiErr := wait(ctx, checker, nil); apiErr != nil {
		return apiErr
	}
	return nil
}

func (c *Client) getClusterByID(ctx context.Context, organizationID string, clusterID string) (*qovery.Cluster, *apierrors.APIError) {
	clusters, res, err := c.api.ClustersApi.
		ListOrganizationCluster(ctx, organizationID).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		return nil, apierrors.NewReadError(apierrors.APIResourceCluster, clusterID, res, err)
	}

	for _, cluster := range clusters.GetResults() {
		if cluster.Id == clusterID {
			return &cluster, nil
		}
	}

	// NOTE: Force status 404 since we didn't find the credential.
	// The status is used to generate the proper error return by the provider.
	res.StatusCode = 404
	return nil, apierrors.NewReadError(apierrors.APIResourceCluster, clusterID, res, err)
}

func (c *Client) updateCluster(ctx context.Context, organizationID string, cluster *qovery.Cluster, params *ClusterUpsertParams) (*ClusterResponse, *apierrors.APIError) {
	if params.ClusterCloudProviderRequest != nil {
		_, res, err := c.api.ClustersApi.
			SpecifyClusterCloudProviderInfo(ctx, organizationID, cluster.Id).
			ClusterCloudProviderInfoRequest(*params.ClusterCloudProviderRequest).
			Execute()
		if err != nil || res.StatusCode >= 400 {
			return nil, apierrors.NewUpdateError(apierrors.APIResourceClusterCloudProvider, cluster.Id, res, err)
		}
	}

	clusterInfo, res, err := c.api.ClustersApi.
		GetOrganizationCloudProviderInfo(ctx, organizationID, cluster.Id).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		return nil, apierrors.NewReadError(apierrors.APIResourceClusterCloudProvider, cluster.Id, res, err)
	}

	var clusterRoutingTable *ClusterRoutingTable
	if len(params.ClusterRoutingTable.Routes) > 0 {
		var apiErr *apierrors.APIError
		clusterRoutingTable, apiErr = c.editClusterRoutingTable(ctx, organizationID, cluster.Id, params.ClusterRoutingTable)
		if apiErr != nil {
			return nil, apiErr
		}
	}

	var advSettings *map[string]interface{}
	var apiErr *apierrors.APIError
	if len(params.ClusterAdvancedSettings) > 0 {
		advSettings, apiErr = c.editClusterAdvancedSettings(ctx, organizationID, cluster.Id, params.ClusterAdvancedSettings)
	} else {
		advSettings, apiErr = c.getClusterAdvancedSettings(ctx, organizationID, cluster.Id)
	}
	if apiErr != nil {
		return nil, apiErr
	}

	clusterStatus, apiErr := c.updateClusterStatus(ctx, organizationID, cluster, params.DesiredState, params.ForceUpdate)
	if apiErr != nil {
		return nil, apiErr
	}
	cluster.Status = clusterStatus.Status

	return &ClusterResponse{
		OrganizationID:         organizationID,
		ClusterResponse:        cluster,
		ClusterRoutingTable:    clusterRoutingTable,
		ClusterInfo:            clusterInfo,
		ClusterAdvancedSetting: advSettings,
	}, nil
}
