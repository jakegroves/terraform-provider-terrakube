# A successful apply in upstream queues a run in downstream.
resource "terrakube_workspace_run_trigger" "example" {
  source_workspace_id      = terrakube_workspace_vcs.upstream.id
  destination_workspace_id = terrakube_workspace_vcs.downstream.id
  template_id              = terrakube_organization_template.plan_apply.id
  enabled                  = true
}
