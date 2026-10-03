package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/harryvince/terraform-provider-cups/internal/cups"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &printerResource{}
	_ resource.ResourceWithConfigure      = &printerResource{}
	_ resource.ResourceWithImportState    = &printerResource{}
	_ resource.ResourceWithValidateConfig = &printerResource{}
)

type printerResource struct{ client *cups.Client }

type printerModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DeviceURI   types.String `tfsdk:"device_uri"`
	Description types.String `tfsdk:"description"`
	Location    types.String `tfsdk:"location"`
}

func (m printerModel) printer() cups.Printer {
	return cups.Printer{Name: m.Name.ValueString(), DeviceURI: m.DeviceURI.ValueString(),
		Description: m.Description.ValueString(), Location: m.Location.ValueString()}
}

func model(p cups.Printer) printerModel {
	return printerModel{ID: types.StringValue(p.Name), Name: types.StringValue(p.Name),
		DeviceURI: types.StringValue(p.DeviceURI), Description: types.StringValue(p.Description), Location: types.StringValue(p.Location)}
}

func (r *printerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_printer"
}

func (r *printerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A persistent driverless IPP printer queue. New queues use CUPS' default paused/rejecting-jobs state; this POC manages configuration only.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, Description: "Queue name within the selected CUPS server.", PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"name":        schema.StringAttribute{Required: true, Description: "Queue name; 1–127 ASCII letters, digits, underscores or hyphens. Changing it replaces the queue.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"device_uri":  schema.StringAttribute{Required: true, Description: "Credential-free IPP/IPPS device URI reachable from the CUPS server. Changing it replaces the queue.", PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"description": schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "Human-readable description. Omission or an empty string clears it."},
			"location":    schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(""), Description: "Human-readable location. Omission or an empty string clears it."},
		},
	}
}

func (r *printerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*cups.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider client", "Expected a CUPS client; report this provider bug.")
		return
	}
	r.client = client
}

func (r *printerResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data printerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, item := range []struct {
		name     string
		value    types.String
		validate func(string) error
	}{{"name", data.Name, cups.ValidateName}, {"device_uri", data.DeviceURI, cups.ValidateDeviceURI}, {"description", data.Description, cups.ValidateText}, {"location", data.Location, cups.ValidateText}} {
		if !item.value.IsUnknown() && !item.value.IsNull() {
			if err := item.validate(item.value.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root(item.name), "Invalid printer configuration", err.Error())
			}
		}
	}
}

func (r *printerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data printerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.Create(ctx, data.printer())
	if p.Name != "" {
		resp.Diagnostics.Append(resp.State.Set(ctx, model(p))...)
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not create CUPS queue", fmt.Sprintf("Queue %q: %s", data.Name.ValueString(), err))
	}
}

func (r *printerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data printerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.Read(ctx, data.ID.ValueString())
	if errors.Is(err, cups.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Could not read CUPS queue", fmt.Sprintf("Queue %q: %s", data.ID.ValueString(), err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model(p))...)
}

func (r *printerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data printerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, err := r.client.Update(ctx, data.printer())
	if err != nil {
		resp.Diagnostics.AddError("Could not update CUPS queue", fmt.Sprintf("Queue %q: %s", data.Name.ValueString(), err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model(p))...)
}

func (r *printerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data printerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Delete(ctx, data.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Could not delete CUPS queue", fmt.Sprintf("Queue %q: %s", data.ID.ValueString(), err))
	}
}

func (r *printerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := cups.ValidateName(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid queue import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
