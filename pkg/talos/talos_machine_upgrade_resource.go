// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package talos

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type talosMachineUpgradeResource struct{}

var (
	_ resource.Resource               = &talosMachineUpgradeResource{}
	_ resource.ResourceWithModifyPlan = &talosMachineUpgradeResource{}
)

type talosMachineUpgradeResourceModelV1 struct {
	ID                  types.String        `tfsdk:"id"`
	Node                types.String        `tfsdk:"node"`
	Endpoint            types.String        `tfsdk:"endpoint"`
	ClientConfiguration clientConfiguration `tfsdk:"client_configuration"`
	Image               types.String        `tfsdk:"image"`
	RebootMode          types.String        `tfsdk:"reboot_mode"`
	Preserve            types.Bool          `tfsdk:"preserve"`
	Stage               types.Bool          `tfsdk:"stage"`
	Force               types.Bool          `tfsdk:"force"`
	Wait                types.Bool          `tfsdk:"wait"`
	Timeouts            timeouts.Value      `tfsdk:"timeouts"`
}

// NewTalosMachineUpgradeResource implements the resource.Resource interface.
func NewTalosMachineUpgradeResource() resource.Resource {
	return &talosMachineUpgradeResource{}
}

func (r *talosMachineUpgradeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_machine_upgrade"
}

func (r *talosMachineUpgradeResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version:     1,
		Description: "The machine upgrade resource allows you to upgrade Talos on a node.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "This is a unique identifier for the machine upgrade operation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"node": schema.StringAttribute{
				Required:    true,
				Description: "The node to upgrade.",
			},
			"endpoint": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The endpoint of the node to upgrade. Defaults to `node`.",
			},
			"client_configuration": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"ca_certificate": schema.StringAttribute{
						Required:    true,
						Description: "The client CA certificate.",
					},
					"client_certificate": schema.StringAttribute{
						Required:    true,
						Description: "The client certificate.",
					},
					"client_key": schema.StringAttribute{
						Required:    true,
						Sensitive:   true,
						Description: "The client key.",
					},
				},
				Required:    true,
				Description: "The Talos client configuration.",
			},
			"image": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The installer image to use for upgrade.",
				Default:     stringdefault.StaticString(GenerateInstallerImage()),
			},
			"reboot_mode": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Reboot mode to use during upgrade (`default`, `powercycle`).",
				Validators: []validator.String{
					stringvalidator.OneOf("default", "powercycle"),
				},
				Default: stringdefault.StaticString("default"),
			},
			"preserve": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Preserve machine data during upgrade.",
				Default:     booldefault.StaticBool(false),
			},
			"stage": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Stage the upgrade and apply after reboot.",
				Default:     booldefault.StaticBool(false),
			},
			"force": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Force upgrade by skipping safety checks.",
				Default:     booldefault.StaticBool(false),
			},
			"wait": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Wait for the upgrade action to complete and node to become ready.",
				Default:     booldefault.StaticBool(true),
			},
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

func (r *talosMachineUpgradeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var state talosMachineUpgradeResourceModelV1

	diags := req.Plan.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	talosClientConfig, err := talosClientTFConfigToTalosClientConfig(
		"dynamic",
		state.ClientConfiguration.CA.ValueString(),
		state.ClientConfiguration.Cert.ValueString(),
		state.ClientConfiguration.Key.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error converting config to talos client config", err.Error())

		return
	}

	createTimeout, diags := state.Timeouts.Create(ctx, 30*time.Minute)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := state.Endpoint.ValueString()
	if endpoint == "" || state.Endpoint.IsNull() || state.Endpoint.IsUnknown() {
		endpoint = state.Node.ValueString()
	}

	if err = machineUpgrade(ctx, endpoint, state.Node.ValueString(), talosClientConfig, machineUpgradeOptions{
		Image:      state.Image.ValueString(),
		RebootMode: state.RebootMode.ValueString(),
		Preserve:   state.Preserve.ValueBool(),
		Stage:      state.Stage.ValueBool(),
		Force:      state.Force.ValueBool(),
		Wait:       state.Wait.ValueBool(),
		Timeout:    createTimeout,
	}); err != nil {
		resp.Diagnostics.AddError("Error upgrading node", err.Error())

		return
	}

	state.ID = basetypes.NewStringValue("machine_upgrade")

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *talosMachineUpgradeResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
}

func (r *talosMachineUpgradeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state talosMachineUpgradeResourceModelV1

	diags := req.Plan.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)

	if diags.HasError() {
		return
	}

	talosClientConfig, err := talosClientTFConfigToTalosClientConfig(
		"dynamic",
		state.ClientConfiguration.CA.ValueString(),
		state.ClientConfiguration.Cert.ValueString(),
		state.ClientConfiguration.Key.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error converting config to talos client config", err.Error())

		return
	}

	updateTimeout, diags := state.Timeouts.Update(ctx, 30*time.Minute)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	endpoint := state.Endpoint.ValueString()
	if endpoint == "" || state.Endpoint.IsNull() || state.Endpoint.IsUnknown() {
		endpoint = state.Node.ValueString()
	}

	if err = machineUpgrade(ctx, endpoint, state.Node.ValueString(), talosClientConfig, machineUpgradeOptions{
		Image:      state.Image.ValueString(),
		RebootMode: state.RebootMode.ValueString(),
		Preserve:   state.Preserve.ValueBool(),
		Stage:      state.Stage.ValueBool(),
		Force:      state.Force.ValueBool(),
		Wait:       state.Wait.ValueBool(),
		Timeout:    updateTimeout,
	}); err != nil {
		resp.Diagnostics.AddError("Error upgrading node", err.Error())

		return
	}

	state.ID = basetypes.NewStringValue("machine_upgrade")

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
}

func (r *talosMachineUpgradeResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
}

func (r *talosMachineUpgradeResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// delete is a no-op
	if req.Plan.Raw.IsNull() {
		return
	}

	var configObj types.Object

	diags := req.Config.Get(ctx, &configObj)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var config talosMachineUpgradeResourceModelV1

	diags = configObj.As(ctx, &config, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	// if either endpoint or node is unknown return early
	if config.Endpoint.IsUnknown() || config.Node.IsUnknown() {
		return
	}

	var planObj types.Object

	diags = req.Plan.Get(ctx, &planObj)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	var planState talosMachineUpgradeResourceModelV1

	diags = planObj.As(ctx, &planState, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	if planState.Endpoint.IsUnknown() || planState.Endpoint.IsNull() {
		diags = resp.Plan.SetAttribute(ctx, path.Root("endpoint"), planState.Node.ValueString())
		resp.Diagnostics.Append(diags...)
	}
}
