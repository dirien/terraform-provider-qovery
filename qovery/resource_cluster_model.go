package qovery

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/qovery/qovery-client-go"

	"github.com/qovery/terraform-provider-qovery/client"
)

const (
	featureKeyVpcSubnet = "vpc_subnet"
	featureIdVpcSubnet  = "VPC_SUBNET"
)

type Cluster struct {
	Id              types.String `tfsdk:"id"`
	OrganizationId  types.String `tfsdk:"organization_id"`
	CredentialsId   types.String `tfsdk:"credentials_id"`
	Name            types.String `tfsdk:"name"`
	CloudProvider   types.String `tfsdk:"cloud_provider"`
	Region          types.String `tfsdk:"region"`
	Description     types.String `tfsdk:"description"`
	KubernetesMode  types.String `tfsdk:"kubernetes_mode"`
	InstanceType    types.String `tfsdk:"instance_type"`
	MinRunningNodes types.Int64  `tfsdk:"min_running_nodes"`
	MaxRunningNodes types.Int64  `tfsdk:"max_running_nodes"`
	Features        types.Object `tfsdk:"features"`
	RoutingTables   types.Set    `tfsdk:"routing_table"`
	State           types.String `tfsdk:"state"`
}

func (c Cluster) hasFeaturesDiff(state *Cluster) bool {
	clusterFeatures := toQoveryClusterFeatures(c.Features)
	if state == nil {
		return len(clusterFeatures) > 0
	}

	stateFeature := toQoveryClusterFeatures(state.Features)
	if len(clusterFeatures) != len(stateFeature) {
		return true
	}

	stateFeaturesByID := make(map[string]string)
	for _, sf := range stateFeature {
		stateFeaturesByID[sf.GetId()] = sf.GetValue()
	}

	for _, cf := range clusterFeatures {
		if stateValue, ok := stateFeaturesByID[cf.GetId()]; !ok || stateValue != cf.GetValue() {
			return true
		}
	}
	return false
}

func (c Cluster) hasRoutingTableDiff(state *Cluster) bool {
	clusterRoutes := toClusterRouteList(c.RoutingTables).toUpsertRequest().Routes
	if state == nil {
		return len(clusterRoutes) > 0
	}

	stateRoutes := toClusterRouteList(state.RoutingTables).toUpsertRequest().Routes
	if len(clusterRoutes) != len(stateRoutes) {
		return true
	}

	stateRoutesByDestination := make(map[string]ClusterRoute)
	for _, sr := range stateRoutes {
		stateRoutesByDestination[sr.Destination] = fromClusterRoute(sr)
	}

	for _, cr := range clusterRoutes {
		stateRoute, ok := stateRoutesByDestination[cr.Destination]
		if !ok {
			return true
		}

		clusterRoute := fromClusterRoute(cr)
		if stateRoute.Description != clusterRoute.Description || stateRoute.Destination != clusterRoute.Destination || stateRoute.Target != clusterRoute.Target {
			return true
		}
	}
	return false
}

func (c Cluster) toUpsertClusterRequest(state *Cluster) (*client.ClusterUpsertParams, error) {
	cloudProvider, err := qovery.NewCloudProviderEnumFromValue(toString(c.CloudProvider))
	if err != nil {
		return nil, err
	}

	kubernetesMode, err := qovery.NewKubernetesEnumFromValue(toString(c.KubernetesMode))
	if err != nil {
		return nil, err
	}

	routingTable := toClusterRouteList(c.RoutingTables)

	var clusterCloudProviderRequest *qovery.ClusterCloudProviderInfoRequest
	if state == nil || c.CredentialsId != state.CredentialsId {
		clusterCloudProviderRequest = &qovery.ClusterCloudProviderInfoRequest{
			CloudProvider: cloudProvider,
			Region:        toStringPointer(c.Region),
			Credentials: &qovery.ClusterCloudProviderInfoCredentials{
				Id:   toStringPointer(c.CredentialsId),
				Name: toStringPointer(c.Name),
			},
		}
	}

	// NOTE: force update clusters if features or routing table have changed
	forceUpdate := c.hasFeaturesDiff(state) || c.hasRoutingTableDiff(state)

	desiredState, err := qovery.NewStateEnumFromValue(toString(c.State))
	if err != nil {
		return nil, err
	}

	return &client.ClusterUpsertParams{
		ClusterCloudProviderRequest: clusterCloudProviderRequest,
		ClusterRequest: qovery.ClusterRequest{
			Name:            toString(c.Name),
			CloudProvider:   *cloudProvider,
			Region:          toString(c.Region),
			Description:     toStringPointer(c.Description),
			Kubernetes:      kubernetesMode,
			InstanceType:    toStringPointer(c.InstanceType),
			MinRunningNodes: toInt32Pointer(c.MinRunningNodes),
			MaxRunningNodes: toInt32Pointer(c.MaxRunningNodes),
			Features:        toQoveryClusterFeatures(c.Features),
		},
		ClusterRoutingTable: routingTable.toUpsertRequest(),
		ForceUpdate:         forceUpdate,
		DesiredState:        *desiredState,
	}, nil
}

func convertResponseToCluster(res *client.ClusterResponse) Cluster {
	routingTable := fromClusterRoutingTable(res.ClusterRoutingTable.Routes)

	return Cluster{
		Id:              fromString(res.ClusterResponse.Id),
		CredentialsId:   fromStringPointer(res.ClusterInfo.Credentials.Id),
		OrganizationId:  fromString(res.OrganizationID),
		Name:            fromString(res.ClusterResponse.Name),
		CloudProvider:   fromClientEnum(res.ClusterResponse.CloudProvider),
		Region:          fromString(res.ClusterResponse.Region),
		Description:     fromStringPointer(res.ClusterResponse.Description),
		KubernetesMode:  fromClientEnumPointer(res.ClusterResponse.Kubernetes),
		InstanceType:    fromStringPointer(res.ClusterResponse.InstanceType),
		MinRunningNodes: fromInt32Pointer(res.ClusterResponse.MinRunningNodes),
		MaxRunningNodes: fromInt32Pointer(res.ClusterResponse.MaxRunningNodes),
		Features:        fromQoveryClusterFeatures(res.ClusterResponse.Features),
		RoutingTables:   routingTable.toTerraformSet(),
		State:           fromClientEnumPointer(res.ClusterResponse.Status),
	}
}

func fromQoveryClusterFeatures(ff []qovery.ClusterFeature) types.Object {
	if ff == nil {
		return types.Object{Null: true}
	}

	attrs := make(map[string]attr.Value)
	attrTypes := make(map[string]attr.Type)
	for _, f := range ff {
		if f.Id == nil {
			continue
		}
		switch *f.Id {
		case featureIdVpcSubnet:
			attrs[featureKeyVpcSubnet] = fromStringPointer(f.GetValue().String)
			attrTypes[featureKeyVpcSubnet] = types.StringType
		}
	}

	if len(attrs) == 0 && len(attrTypes) == 0 {
		return types.Object{Unknown: true}
	}

	return types.Object{
		Attrs:     attrs,
		AttrTypes: attrTypes,
	}
}

func toQoveryClusterFeatures(f types.Object) []qovery.ClusterRequestFeaturesInner {
	if f.Null || f.Unknown {
		return nil
	}

	features := make([]qovery.ClusterRequestFeaturesInner, 0, len(f.Attrs))
	if _, ok := f.Attrs[featureKeyVpcSubnet]; ok {
		features = append(features, qovery.ClusterRequestFeaturesInner{
			Id:    stringAsPointer(featureIdVpcSubnet),
			Value: *qovery.NewNullableString(toStringPointer(f.Attrs[featureKeyVpcSubnet].(types.String))),
		})
	}

	return features
}
