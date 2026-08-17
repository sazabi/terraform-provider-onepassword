package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/sazabi/terraform-provider-onepassword/internal/opclient"
)

var (
	_ resource.Resource                = &vaultGroupAccessResource{}
	_ resource.ResourceWithConfigure   = &vaultGroupAccessResource{}
	_ resource.ResourceWithImportState = &vaultGroupAccessResource{}
)

// NewVaultGroupAccessResource is the onepassword_vault_group_access factory.
func NewVaultGroupAccessResource() resource.Resource {
	return &vaultGroupAccessResource{}
}

type vaultGroupAccessResource struct {
	client *opclient.Client
}

type vaultGroupAccessModel struct {
	ID          types.String `tfsdk:"id"`
	VaultID     types.String `tfsdk:"vault_id"`
	Group       types.String `tfsdk:"group"`
	Permissions types.Set    `tfsdk:"permissions"`
}

func (r *vaultGroupAccessResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault_group_access"
}

func (r *vaultGroupAccessResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Grants a 1Password group access to a vault at a set of permissions. Bind this to a " +
			"directory-synced group (by name or UUID). Note: granting a group on an *existing* vault requires the " +
			"human admin `op signin` session — a Service Account can only manage vaults it created.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Synthetic id, `<vault_id>:<group>`.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"vault_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The vault UUID to grant access to.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The group to grant, by name or UUID (typically a directory-synced group).",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"permissions": schema.SetAttribute{
				Required:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "The permission tokens to grant, e.g. `allow_viewing`, `allow_editing`, `allow_managing`.",
			},
		},
	}
}

func (r *vaultGroupAccessResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// permissionsOf converts the set attribute to a sorted []string.
func permissionsOf(set types.Set) []string {
	var perms []string
	for _, e := range set.Elements() {
		if s, ok := e.(types.String); ok {
			perms = append(perms, s.ValueString())
		}
	}
	sort.Strings(perms)
	return perms
}

func (r *vaultGroupAccessResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan vaultGroupAccessModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms := permissionsOf(plan.Permissions)
	if err := r.client.GrantVaultGroup(ctx, plan.VaultID.ValueString(), plan.Group.ValueString(), perms); err != nil {
		resp.Diagnostics.AddError("Error granting vault access", err.Error())
		return
	}

	plan.ID = types.StringValue(grantID(plan.VaultID.ValueString(), plan.Group.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *vaultGroupAccessResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state vaultGroupAccessModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	g, err := r.client.FindVaultGroup(ctx, state.VaultID.ValueString(), state.Group.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading vault group access", err.Error())
		return
	}
	if g == nil {
		// Grant gone remotely -> drop from state.
		resp.State.RemoveResource(ctx)
		return
	}

	permSet, d := types.SetValueFrom(ctx, types.StringType, normalizePerms(g.Permissions))
	resp.Diagnostics.Append(d...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Permissions = permSet
	state.ID = types.StringValue(grantID(state.VaultID.ValueString(), state.Group.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *vaultGroupAccessResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan vaultGroupAccessModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// vault_id and group force replacement, so only permissions can change here.
	// Revoke the whole grant, then re-grant the exact declared set — the
	// simplest way to make the declared permissions authoritative.
	vaultID := plan.VaultID.ValueString()
	group := plan.Group.ValueString()
	if err := r.client.RevokeVaultGroup(ctx, vaultID, group, nil); err != nil {
		resp.Diagnostics.AddError("Error updating vault access (revoke step)", err.Error())
		return
	}
	perms := permissionsOf(plan.Permissions)
	if err := r.client.GrantVaultGroup(ctx, vaultID, group, perms); err != nil {
		resp.Diagnostics.AddError("Error updating vault access (grant step)", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *vaultGroupAccessResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state vaultGroupAccessModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.RevokeVaultGroup(ctx, state.VaultID.ValueString(), state.Group.ValueString(), nil); err != nil {
		resp.Diagnostics.AddError("Error revoking vault access", err.Error())
		return
	}
}

func (r *vaultGroupAccessResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import id format: <vault_id>:<group>
	vaultID, group, ok := strings.Cut(req.ID, ":")
	if !ok || vaultID == "" || group == "" {
		resp.Diagnostics.AddError("Invalid import id", "expected format <vault_id>:<group>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("vault_id"), vaultID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("group"), group)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), grantID(vaultID, group))...)
}

func grantID(vaultID, group string) string {
	return vaultID + ":" + group
}

// normalizePerms sorts permission tokens so state comparison is stable.
func normalizePerms(perms []string) []string {
	out := append([]string(nil), perms...)
	sort.Strings(out)
	return out
}
