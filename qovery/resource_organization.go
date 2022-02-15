package qovery

import (
	"context"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/qovery/qovery-client-go"

	"terraform-provider-qovery/qovery/apierror"
	"terraform-provider-qovery/qovery/descriptions"
	"terraform-provider-qovery/qovery/validators"
)

const organizationAPIResource = "organization"

var organizationPlans = []string{"FREE", "PROFESSIONAL", "BUSINESS"}

type organizationResourceType struct{}

func (r organizationResourceType) GetSchema(_ context.Context) (tfsdk.Schema, diag.Diagnostics) {
	return tfsdk.Schema{
		Description: "Provides a Qovery organization resource. This can be used to create and manage Qovery organizations.",
		Attributes: map[string]tfsdk.Attribute{
			"id": {
				Description: "Id of the organization.",
				Type:        types.StringType,
				Computed:    true,
			},
			"name": {
				Description: "Name of the organization.",
				Type:        types.StringType,
				Required:    true,
			},
			"plan": {
				Description: descriptions.NewStringEnumDescription(
					"Plan of the organization.",
					organizationPlans,
					nil,
				),
				Type:     types.StringType,
				Required: true,
				Validators: []tfsdk.AttributeValidator{
					validators.StringEnumValidator{Enum: organizationPlans},
				},
			},
			"description": {
				Description: "Description of the organization.",
				Type:        types.StringType,
				Optional:    true,
			},
		},
	}, nil
}

func (r organizationResourceType) NewResource(_ context.Context, p tfsdk.Provider) (tfsdk.Resource, diag.Diagnostics) {
	return organizationResource{
		client: p.(*provider).GetClient(),
	}, nil
}

type organizationResource struct {
	client *qovery.APIClient
}

// Create qovery organization resource
func (r organizationResource) Create(ctx context.Context, req tfsdk.CreateResourceRequest, resp *tfsdk.CreateResourceResponse) {
	// Retrieve values from plan
	var plan Organization
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Create new organization
	organization, res, err := r.client.OrganizationMainCallsApi.
		CreateOrganization(ctx).
		OrganizationRequest(plan.toCreateOrganizationRequest()).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		apiErr := organizationCreateAPIError(plan.Name.Value, res, err)
		resp.Diagnostics.AddError(apiErr.Summary(), apiErr.Detail())
		return
	}

	// Initialize state values
	state := convertResponseToOrganization(organization)
	tflog.Trace(ctx, "created organization", "organization_id", state.Id.Value)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Read qovery organization resource
func (r organizationResource) Read(ctx context.Context, req tfsdk.ReadResourceRequest, resp *tfsdk.ReadResourceResponse) {
	// Get current state
	var state Organization
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get organization from API
	organization, res, err := r.client.OrganizationMainCallsApi.
		GetOrganization(ctx, state.Id.Value).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		apiErr := organizationReadAPIError(state.Id.Value, res, err)
		resp.Diagnostics.AddError(apiErr.Summary(), apiErr.Detail())
		return
	}

	// Refresh state values
	state = convertResponseToOrganization(organization)
	tflog.Trace(ctx, "read organization", "organization_id", state.Id.Value)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update qovery organization resource
func (r organizationResource) Update(ctx context.Context, req tfsdk.UpdateResourceRequest, resp *tfsdk.UpdateResourceResponse) {
	// Get plan and current state
	var plan, state Organization
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Update organization in backend
	organization, res, err := r.client.OrganizationMainCallsApi.
		EditOrganization(ctx, state.Id.Value).
		OrganizationEditRequest(plan.toUpdateOrganizationRequest()).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		apiErr := organizationUpdateAPIError(state.Id.Value, res, err)
		resp.Diagnostics.AddError(apiErr.Summary(), apiErr.Detail())
		return
	}

	// Update state values
	state = convertResponseToOrganization(organization)
	tflog.Trace(ctx, "updated organization", "organization_id", state.Id.Value)

	// Set state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete qovery organization resource
func (r organizationResource) Delete(ctx context.Context, req tfsdk.DeleteResourceRequest, resp *tfsdk.DeleteResourceResponse) {
	// Get current state
	var state Organization
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete organization
	res, err := r.client.OrganizationMainCallsApi.
		DeleteOrganization(ctx, state.Id.Value).
		Execute()
	if err != nil || res.StatusCode >= 400 {
		apiErr := organizationDeleteAPIError(state.Id.Value, res, err)
		resp.Diagnostics.AddError(apiErr.Summary(), apiErr.Detail())
		return
	}

	tflog.Trace(ctx, "deleted organization", "organization_id", state.Id.Value)

	// Remove organization from state
	resp.State.RemoveResource(ctx)
}

// ImportState imports a qovery organization resource using its id
func (r organizationResource) ImportState(ctx context.Context, req tfsdk.ImportResourceStateRequest, resp *tfsdk.ImportResourceStateResponse) {
	tfsdk.ResourceImportStatePassthroughID(ctx, tftypes.NewAttributePath().WithAttributeName("id"), req, resp)
}

func organizationCreateAPIError(organizationName string, res *http.Response, err error) *apierror.APIError {
	return apierror.New(organizationAPIResource, organizationName, apierror.Create, res, err)
}

func organizationReadAPIError(organizationID string, res *http.Response, err error) *apierror.APIError {
	return apierror.New(organizationAPIResource, organizationID, apierror.Read, res, err)
}

func organizationUpdateAPIError(organizationID string, res *http.Response, err error) *apierror.APIError {
	return apierror.New(organizationAPIResource, organizationID, apierror.Update, res, err)
}

func organizationDeleteAPIError(organizationID string, res *http.Response, err error) *apierror.APIError {
	return apierror.New(organizationAPIResource, organizationID, apierror.Delete, res, err)
}
