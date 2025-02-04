package qoveryapi

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qovery/qovery-client-go"

	"github.com/qovery/terraform-provider-qovery/internal/domain/apierrors"
	"github.com/qovery/terraform-provider-qovery/internal/domain/job"
)

// Ensure jobQoveryAPI defined types fully satisfy the job.Repository interface.
var _ job.Repository = jobQoveryAPI{}

// jobQoveryAPI implements the interface job.Repository.
type jobQoveryAPI struct {
	client *qovery.APIClient
}

// newJobQoveryAPI return a new instance of a job.Repository that uses Qovery's API.
func newJobQoveryAPI(client *qovery.APIClient) (job.Repository, error) {
	if client == nil {
		return nil, ErrInvalidQoveryAPIClient
	}

	return &jobQoveryAPI{
		client: client,
	}, nil
}

// Create calls Qovery's API to create a job for an organization using the given organizationID and request.
func (c jobQoveryAPI) Create(ctx context.Context, environmentID string, request job.UpsertRepositoryRequest) (*job.Job, error) {
	req, err := newQoveryJobRequestFromDomain(request)
	if err != nil {
		return nil, errors.Wrap(err, job.ErrInvalidJobUpsertRequest.Error())
	}

	newJob, resp, err := c.client.JobsApi.
		CreateJob(ctx, environmentID).
		JobRequest(*req).
		Execute()
	if err != nil || resp.StatusCode >= 400 {
		return nil, apierrors.NewCreateApiError(apierrors.ApiResourceJob, request.Name, resp, err)
	}

	// Attach job to deployment stage
	if len(request.DeploymentStageID) > 0 {
		_, response, err := c.client.DeploymentStageMainCallsApi.AttachServiceToDeploymentStage(ctx, request.DeploymentStageID, newJob.Id).Execute()
		if err != nil || response.StatusCode >= 400 {
			return nil, apierrors.NewCreateApiError(apierrors.ApiResourceJob, request.Name, resp, err)
		}
	}

	// Get job deployment stage
	deploymentStage, resp, err := c.client.DeploymentStageMainCallsApi.GetServiceDeploymentStage(ctx, newJob.Id).Execute()
	if err != nil || resp.StatusCode >= 400 {
		return nil, apierrors.NewCreateApiError(apierrors.ApiResourceJob, newJob.Id, resp, err)
	}

	return newDomainJobFromQovery(newJob, deploymentStage.Id)
}

// Get calls Qovery's API to retrieve a job using the given jobID.
func (c jobQoveryAPI) Get(ctx context.Context, jobID string) (*job.Job, error) {
	job, resp, err := c.client.JobMainCallsApi.
		GetJob(ctx, jobID).
		Execute()
	if err != nil || resp.StatusCode >= 400 {
		return nil, apierrors.NewReadApiError(apierrors.ApiResourceJob, jobID, resp, err)
	}

	// Get job deployment stage
	deploymentStage, resp, err := c.client.DeploymentStageMainCallsApi.GetServiceDeploymentStage(ctx, job.Id).Execute()
	if err != nil || resp.StatusCode >= 400 {
		return nil, apierrors.NewCreateApiError(apierrors.ApiResourceJob, job.Id, resp, err)
	}

	return newDomainJobFromQovery(job, deploymentStage.Id)
}

// Update calls Qovery's API to update a job using the given jobID and request.
func (c jobQoveryAPI) Update(ctx context.Context, jobID string, request job.UpsertRepositoryRequest) (*job.Job, error) {
	req, err := newQoveryJobRequestFromDomain(request)
	if err != nil {
		return nil, errors.Wrap(err, job.ErrInvalidJobUpsertRequest.Error())
	}

	job, resp, err := c.client.JobMainCallsApi.
		EditJob(ctx, jobID).
		JobRequest(*req).
		Execute()
	if err != nil || resp.StatusCode >= 400 {
		return nil, apierrors.NewUpdateApiError(apierrors.ApiResourceJob, jobID, resp, err)
	}

	// Attach job to deployment stage
	if len(request.DeploymentStageID) > 0 {
		_, response, err := c.client.DeploymentStageMainCallsApi.AttachServiceToDeploymentStage(ctx, request.DeploymentStageID, job.Id).Execute()
		if err != nil || response.StatusCode >= 400 {
			return nil, apierrors.NewCreateApiError(apierrors.ApiResourceJob, request.Name, resp, err)
		}
	}

	// Get job deployment stage
	deploymentStage, resp, err := c.client.DeploymentStageMainCallsApi.GetServiceDeploymentStage(ctx, job.Id).Execute()
	if err != nil || resp.StatusCode >= 400 {
		return nil, apierrors.NewCreateApiError(apierrors.ApiResourceJob, job.Id, resp, err)
	}

	return newDomainJobFromQovery(job, deploymentStage.Id)
}

// Delete calls Qovery's API to deletes a job using the given jobID.
func (c jobQoveryAPI) Delete(ctx context.Context, jobID string) error {
	_, resp, err := c.client.JobMainCallsApi.
		GetJob(ctx, jobID).
		Execute()
	if err != nil || resp.StatusCode >= 400 {
		if resp.StatusCode == 404 {
			// if the job is not found, then it has already been deleted
			return nil
		}
		return apierrors.NewDeleteApiError(apierrors.ApiResourceJob, jobID, resp, err)
	}

	resp, err = c.client.JobMainCallsApi.
		DeleteJob(ctx, jobID).
		Execute()
	if err != nil || resp.StatusCode >= 300 {
		return apierrors.NewDeleteApiError(apierrors.ApiResourceJob, jobID, resp, err)
	}

	return nil
}
