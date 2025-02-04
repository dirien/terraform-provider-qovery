# qovery_deployment (Resource)

Provides a Qovery deployment stage resource. This can be used to create and manage Qovery deployment stages.


## Example
```terraform
resource "qovery_deployment" "my_deployment" {
  # Required
  environment_id = qovery_environment.my_environment.id
  desired_state  = "RUNNING"
  version        = "random_uuid_to_force_retrigger_terraform_apply"

  depends_on = [
    qovery_application.my_application,
    qovery_database.my_database,
    qovery_container.my_container,
  ]
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `desired_state` (String) Desired state of the deployment.
	- Can be: `RESTARTED`, `RUNNING`, `STOPPED`.
- `environment_id` (String) Id of the environment.

### Optional

- `id` (String) Id of the deployment
- `version` (String) Version to force trigger a deployment when desired_state doesn't change (e.g redeploy a deployment having the 'RUNNING' state)
