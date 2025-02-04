# qovery_deployment_stage (Resource)

Provides a Qovery deployment stage resource. This can be used to create and manage Qovery deployment stages.


## Example
```terraform
resource "qovery_deployment_stage" "my_deployment_stage" {
  # Required
  environment_id = qovery_environment.my_environment.id
  name           = "MyDeploymentStage"

  # Optional
  description = ""
  move_after  = qovery_deployment_stage.first_deployment_stage.id
  move_before = qovery_deployment_stage.third_deployment_stage.id

  depends_on = [
    qovery_environment.my_environment
  ]
}
```

You can find complete examples within these repositories:

* [Deploy services with a specific order](https://github.com/Qovery/terraform-examples/tree/main/examples/deploy-services-with-a-specific-order)
<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `environment_id` (String) Id of the environment.
- `name` (String) Name of the deployment stage.

### Optional

- `description` (String) Description of the deployment stage.
- `move_after` (String) Move the current deployment stage after the target deployment stage
- `move_before` (String) Move the current deployment stage before the target deployment stage

### Read-Only

- `id` (String) Id of the deployment stage.
## Import
```shell
terraform import qovery_deployment_stage.my_deployment_stage "<deployment_stage_id>"
```