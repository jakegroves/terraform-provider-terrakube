package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func objectField(t *testing.T, object map[string]any, name string) map[string]any {
	t.Helper()

	value, ok := object[name]
	if !ok {
		t.Fatalf("expected object to contain %q: %#v", name, object)
	}
	field, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %q to be an object, got %T", name, value)
	}
	return field
}

func TestWorkspaceRunTriggerResourceSchema(t *testing.T) {
	ctx := context.Background()
	r := &WorkspaceRunTriggerResource{}
	var response resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &response)
	if response.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %v", response.Diagnostics)
	}

	for _, name := range []string{"source_workspace_id", "destination_workspace_id"} {
		attribute, ok := response.Schema.Attributes[name]
		if !ok {
			t.Fatalf("expected schema to define %q", name)
		}
		stringAttribute, ok := attribute.(schema.StringAttribute)
		if !ok || !stringAttribute.Required {
			t.Errorf("expected %q to be a required StringAttribute", name)
		}
		foundRequiresReplace := false
		for _, modifier := range stringAttribute.PlanModifiers {
			if strings.Contains(modifier.Description(ctx), "destroy and recreate the resource") {
				foundRequiresReplace = true
			}
		}
		if !foundRequiresReplace {
			t.Errorf("expected %q to require replacement when changed", name)
		}
	}

	enabled, ok := response.Schema.Attributes["enabled"].(schema.BoolAttribute)
	if !ok || !enabled.Optional || !enabled.Computed || enabled.Default == nil {
		t.Error("expected enabled to be optional, computed, and have a default")
	}
}

func TestWorkspaceRunTriggerRequestBodySendsNullTemplate(t *testing.T) {
	model := WorkspaceRunTriggerResourceModel{
		SourceWorkspaceID:      types.StringValue("source-1"),
		DestinationWorkspaceID: types.StringValue("destination-1"),
		TemplateID:             types.StringNull(),
		Enabled:                types.BoolValue(true),
	}
	body, err := workspaceRunTriggerUpdateRequestBody(model, "trigger-1")
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshalling request body: %v", err)
	}
	data := objectField(t, payload, "data")
	if data["type"] != "runTrigger" || data["id"] != "trigger-1" {
		t.Errorf("unexpected JSON:API resource identifier: %#v", data)
	}
	relationships := objectField(t, data, "relationships")
	if objectField(t, relationships, "template")["data"] != nil {
		t.Errorf("expected omitted template_id to be encoded as a null relationship, got %s", body)
	}
	if _, exists := relationships["sourceWorkspace"]; exists {
		t.Errorf("PATCH must not resend immutable sourceWorkspace, got %s", body)
	}
	if _, exists := relationships["destinationWorkspace"]; exists {
		t.Errorf("PATCH must not resend immutable destinationWorkspace, got %s", body)
	}
}

func TestWorkspaceRunTriggerCreateRequestBodyIncludesWorkspaceRelationships(t *testing.T) {
	model := WorkspaceRunTriggerResourceModel{
		SourceWorkspaceID:      types.StringValue("source-1"),
		DestinationWorkspaceID: types.StringValue("destination-1"),
		Enabled:                types.BoolValue(true),
	}
	body, err := workspaceRunTriggerCreateRequestBody(model)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshalling request body: %v", err)
	}
	relationships := objectField(t, objectField(t, payload, "data"), "relationships")
	for relationship, id := range map[string]string{"sourceWorkspace": "source-1", "destinationWorkspace": "destination-1"} {
		identifier := objectField(t, objectField(t, relationships, relationship), "data")
		if identifier["type"] != "workspace" || identifier["id"] != id {
			t.Errorf("unexpected %s relationship: %#v", relationship, identifier)
		}
	}
}
