package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sazabi/terraform-provider-onepassword/internal/opclient"
)

var (
	_ resource.Resource                = &vaultResource{}
	_ resource.ResourceWithConfigure   = &vaultResource{}
	_ resource.ResourceWithImportState = &vaultResource{}
)

// NewVaultResource is the onepassword_vault resource factory.
func NewVaultResource() resource.Resource {
	return &vaultResource{}
}

type vaultResource struct {
	client *opclient.Client
}

type vaultResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Icon             types.String `tfsdk:"icon"`
	DeleteProtection types.Bool   `tfsdk:"delete_protection"`
}

func (r *vaultResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (r *vaultResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A 1Password vault. Managed via the `op` CLI. Deleting a vault is guarded by " +
			"`delete_protection` (on by default); pair with a `lifecycle { prevent_destroy = true }` block for a " +
			"second, plan-time guard.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The vault UUID assigned by 1Password.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The vault name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "The vault description.",
			},
			"icon": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "The vault icon name (write-only on create/update; not refreshed from the API, " +
					"so it never shows spurious drift).",
			},
			"delete_protection": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				MarkdownDescription: "When true (the default), `terraform destroy` is refused for this vault. Set to " +
					"false in a first apply, then destroy in a second — the deliberate two-step for a destructive op.",
			},
		},
	}
}

func (r *vaultResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*opclient.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *opclient.Client, got %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *vaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	v, err := r.client.CreateVault(ctx, plan.Name.ValueString(), plan.Description.ValueString(), plan.Icon.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error creating vault", err.Error())
		return
	}

	plan.ID = types.StringValue(v.ID)
	plan.Name = types.StringValue(v.Name)
	// description/icon retained from plan; op create output may omit them.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *vaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vaultResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	v, err := r.client.GetVault(ctx, state.ID.ValueString())
	if err != nil {
		// Vault gone remotely -> drop from state so Terraform plans a recreate.
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(v.Name)
	// Only overwrite description when the API returns one; otherwise keep prior
	// state to avoid flapping empty <-> set on drivers that omit it.
	if v.Description != "" {
		state.Description = types.StringValue(v.Description)
	}
	// icon is intentionally not refreshed (see schema).
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vaultResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.EditVault(ctx, plan.ID.ValueString(), plan.Name.ValueString(), plan.Description.ValueString(), plan.Icon.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating vault", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *vaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vaultResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if state.DeleteProtection.ValueBool() {
		resp.Diagnostics.AddError(
			"Vault delete protection is enabled",
			fmt.Sprintf("Vault %q (%s) has delete_protection = true. Set delete_protection = false and apply, then destroy.",
				state.Name.ValueString(), state.ID.ValueString()),
		)
		return
	}

	if err := r.client.DeleteVault(ctx, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting vault", err.Error())
		return
	}
}

func (r *vaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
