package provider

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/thoas/go-funk"
)

var _ resource.Resource = &LDAPObjectResource{}
var _ resource.ResourceWithImportState = &LDAPObjectResource{}
var _ resource.ResourceWithModifyPlan = &LDAPObjectResource{}
var _ resource.ResourceWithConfigure = &LDAPObjectResource{}

func NewLDAPObjectResource() resource.Resource {
	return &LDAPObjectResource{}
}

type LDAPObjectResource struct {
	conn *ldap.Conn
}

type LDAPObjectResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DN            types.String `tfsdk:"dn"`
	ObjectClasses types.List   `tfsdk:"object_classes"`
	Attributes    types.Map    `tfsdk:"attributes"`
	IgnoreChanges types.List   `tfsdk:"ignore_changes"`
}

func (L *LDAPObjectResource) Metadata(_ context.Context, request resource.MetadataRequest, response *resource.MetadataResponse) {
	response.TypeName = request.ProviderTypeName + "_object"
}

func (L *LDAPObjectResource) Schema(_ context.Context, _ resource.SchemaRequest, response *resource.SchemaResponse) {
	response.Schema = schema.Schema{
		MarkdownDescription: "Generic LDAP object resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Resource identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dn": schema.StringAttribute{
				MarkdownDescription: "DN of this ldap object",
				Required:            true,
			},
			"object_classes": schema.ListAttribute{
				MarkdownDescription: "A list of classes this object implements",
				ElementType:         types.StringType,
				Required:            true,
			},
			"attributes": schema.MapAttribute{
				MarkdownDescription: "The definition of an attribute, the name defines the type of the attribute",
				Optional:            true,
				ElementType:         types.ListType{ElemType: types.StringType},
			},
			"ignore_changes": schema.ListAttribute{
				MarkdownDescription: "A list of types for which changes are ignored",
				Optional:            true,
				ElementType:         types.StringType,
			},
		},
	}
}

func (L *LDAPObjectResource) Configure(_ context.Context, request resource.ConfigureRequest, response *resource.ConfigureResponse) {
	if request.ProviderData == nil {
		return
	}

	if conn, ok := request.ProviderData.(*ldap.Conn); !ok {
		response.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ldap.Conn, got: %T. Please report this issue to the provider developers.", request.ProviderData),
		)

		return
	} else {
		L.conn = conn
	}
}

func (L *LDAPObjectResource) Create(ctx context.Context, request resource.CreateRequest, response *resource.CreateResponse) {
	var data *LDAPObjectResourceModel
	response.Diagnostics.Append(request.Plan.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	if err := L.addLdapEntry(ctx, data, &response.Diagnostics); err != nil {
		response.Diagnostics.AddError(
			"Can not add resource",
			fmt.Sprintf("LDAP server reported: %s", err),
		)
		return
	}
	data.ID = data.DN
	response.Diagnostics.Append(response.State.Set(ctx, &data)...)
}

func (L *LDAPObjectResource) Read(ctx context.Context, request resource.ReadRequest, response *resource.ReadResponse) {
	var data *LDAPObjectResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &data)...)
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading entry", map[string]interface{}{"dn": data.DN.ValueString()})
	if entry, err := GetEntry(L.conn, data.DN.ValueString()); err != nil {
		response.Diagnostics.AddError(
			"Can not read entry",
			err.Error(),
		)
	} else {
		response.State.SetAttribute(ctx, path.Root("dn"), entry.DN)
		ctx = MaskAttributesFromArray(ctx, entry.Attributes)
		for _, attribute := range entry.Attributes {
			if attribute.Name == "objectClass" {
				response.State.SetAttribute(ctx, path.Root("object_classes"), attribute.Values)
			} else if !L.isIgnored(ctx, attribute.Name, data, response.Diagnostics) {
				response.State.SetAttribute(ctx, path.Root("attributes").AtMapKey(attribute.Name), attribute.Values)
			}
		}

		tflog.Debug(ctx, "Read entry", map[string]interface{}{"entry": ToLDIF(entry)})
	}
}

func (L *LDAPObjectResource) Update(ctx context.Context, request resource.UpdateRequest, response *resource.UpdateResponse) {
	var stateData *LDAPObjectResourceModel
	var planData *LDAPObjectResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)
	response.Diagnostics.Append(request.Plan.Get(ctx, &planData)...)
	if response.Diagnostics.HasError() {
		return
	}

	// Recreate object if DN changed
	if stateData.DN.ValueString() != planData.DN.ValueString() {
		tflog.Warn(ctx, "Recreating entry because the DN changed", map[string]interface{}{
			"oldDn": stateData.DN.ValueString(),
			"dn":    planData.DN.ValueString(),
		})

		if err := L.conn.Del(ldap.NewDelRequest(stateData.DN.ValueString(), []ldap.Control{})); err != nil {
			response.Diagnostics.AddError(
				"Can not delete old DN entry",
				fmt.Sprintf("Trying to delete entry of old DN returned: %s", err),
			)
			return
		}
		if err := L.addLdapEntry(ctx, planData, &response.Diagnostics); err != nil {
			response.Diagnostics.AddError(
				"Can not add resource",
				fmt.Sprintf("LDAP server reported: %s", err),
			)
			return
		}
	} else {
		r := ldap.NewModifyRequest(planData.DN.ValueString(), []ldap.Control{})

		var stateObjectClasses []string
		response.Diagnostics.Append(stateData.ObjectClasses.ElementsAs(ctx, &stateObjectClasses, false)...)
		var planObjectClasses []string
		response.Diagnostics.Append(planData.ObjectClasses.ElementsAs(ctx, &planObjectClasses, false)...)

		var classesToAdd []string
		for _, class := range planObjectClasses {
			if funk.IndexOf(stateObjectClasses, class) == -1 {
				classesToAdd = append(classesToAdd, class)
			}
		}

		if len(classesToAdd) > 0 {
			r.Add("objectClass", classesToAdd)
		}

		var stateAttributes map[string][]string
		response.Diagnostics.Append(stateData.Attributes.ElementsAs(ctx, &stateAttributes, false)...)
		var planAttributes map[string][]string
		response.Diagnostics.Append(planData.Attributes.ElementsAs(ctx, &planAttributes, false)...)

		ctx = MaskAttributes(ctx, stateAttributes)
		for attributeType, stateValues := range stateAttributes {
			if L.isIgnored(ctx, attributeType, stateData, response.Diagnostics) {
				continue
			}
			// state attribute is in the plan, compare the values
			if planValues, exists := planAttributes[attributeType]; exists {
				valuesChanged := false
				for _, stateValue := range stateValues {
					if !funk.ContainsString(planValues, stateValue) {
						valuesChanged = true
					}
				}
				for _, planValue := range planValues {
					if !funk.ContainsString(stateValues, planValue) {
						valuesChanged = true
					}
				}
				if valuesChanged {
					tflog.Debug(ctx, "Changing attribute", map[string]interface{}{
						"type":   attributeType,
						"values": planValues,
					})
					r.Replace(attributeType, planValues)
				}
			} else {
				tflog.Debug(ctx, "Removing attribute", map[string]interface{}{
					"type": attributeType,
				})
				r.Delete(attributeType, []string{})
			}
		}
		for attributeType, values := range planAttributes {
			if L.isIgnored(ctx, attributeType, planData, response.Diagnostics) {
				continue
			}
			if _, exists := stateAttributes[attributeType]; !exists {
				tflog.Debug(ctx, "Adding attribute", map[string]interface{}{
					"type": attributeType,
				})
				r.Add(attributeType, values)
			}
		}
		if err := L.conn.Modify(r); err != nil {
			response.Diagnostics.AddError(
				"Can not modify entry",
				fmt.Sprintf("LDAP server reported: %s", err),
			)
			return
		}
	}
	planData.ID = planData.DN
	response.Diagnostics.Append(response.State.Set(ctx, &planData)...)
}

func (L *LDAPObjectResource) Delete(ctx context.Context, request resource.DeleteRequest, response *resource.DeleteResponse) {
	var stateData *LDAPObjectResourceModel
	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)
	if response.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting entry", map[string]interface{}{"dn": stateData.DN.ValueString()})
	if err := L.conn.Del(ldap.NewDelRequest(stateData.DN.ValueString(), []ldap.Control{})); err != nil {
		response.Diagnostics.AddError(
			"Can not delete entry",
			fmt.Sprintf("Trying to delete entry returned: %s", err),
		)
		return
	}
}

func (L *LDAPObjectResource) ImportState(ctx context.Context, request resource.ImportStateRequest, response *resource.ImportStateResponse) {
	tflog.Info(ctx, "Importing entry", map[string]interface{}{"dn": request.ID})
	if entry, err := GetEntry(L.conn, request.ID); err != nil {
		response.Diagnostics.AddError(
			"Can not read entry",
			err.Error(),
		)
	} else {
		resource.ImportStatePassthroughID(ctx, path.Root("id"), request, response)
		ctx = MaskAttributesFromArray(ctx, entry.Attributes)
		response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("dn"), entry.DN)...)
		for _, attribute := range entry.Attributes {
			if attribute.Name == "objectClass" {
				response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("object_classes"), attribute.Values)...)
			} else {
				response.Diagnostics.Append(response.State.SetAttribute(ctx, path.Root("attributes").AtMapKey(attribute.Name), attribute.Values)...)
			}
		}
		tflog.Debug(ctx, "Imported entry", map[string]interface{}{
			"entry": ToLDIF(entry),
		})
	}
}

func (L *LDAPObjectResource) ModifyPlan(ctx context.Context, request resource.ModifyPlanRequest, response *resource.ModifyPlanResponse) {
	var stateData *LDAPObjectResourceModel
	var planData *LDAPObjectResourceModel

	response.Diagnostics.Append(request.State.Get(ctx, &stateData)...)
	response.Diagnostics.Append(request.Plan.Get(ctx, &planData)...)
	if stateData == nil || planData == nil {
		// don't ignore any attributes on create and delete
		return
	}

	if stateData != nil && planData != nil && stateData.DN != planData.DN {
		response.Diagnostics.Append(response.Plan.SetAttribute(ctx, path.Root("id"), types.StringUnknown())...)
		if response.Diagnostics.HasError() {
			return
		}
	}

	var planAttributes map[string][]string
	response.Diagnostics.Append(planData.Attributes.ElementsAs(ctx, &planAttributes, false)...)
	var stateAttributes map[string][]string
	response.Diagnostics.Append(stateData.Attributes.ElementsAs(ctx, &stateAttributes, false)...)
	if response.Diagnostics.HasError() {
		return
	}

	for attributeType := range planAttributes {
		if L.isIgnored(ctx, attributeType, planData, response.Diagnostics) {
			response.Plan.SetAttribute(ctx, path.Root("attributes").AtMapKey(attributeType), stateAttributes[attributeType])
		}
	}

	for attributeType := range stateAttributes {
		if L.isIgnored(ctx, attributeType, planData, response.Diagnostics) {
			// Re-add attributes to the plan that were ignored and removed to not manage them
			response.Plan.SetAttribute(ctx, path.Root("attributes").AtMapKey(attributeType), stateAttributes[attributeType])
		}
	}
}

func (L *LDAPObjectResource) addLdapEntry(ctx context.Context, data *LDAPObjectResourceModel, diagnostics *diag.Diagnostics) error {
	var objectClasses []string
	diagnostics.Append(data.ObjectClasses.ElementsAs(ctx, &objectClasses, false)...)
	if diagnostics.HasError() {
		return errors.New("error converting data")
	}

	var attributes map[string][]string
	diagnostics.Append(data.Attributes.ElementsAs(ctx, &attributes, false)...)
	if diagnostics.HasError() {
		return errors.New("error converting data")
	}

	tflog.Info(ctx, "Adding new item", map[string]interface{}{
		"dn":          data.DN.ValueString(),
		"objectClass": objectClasses,
		"attributes":  attributes,
	})
	a := ldap.NewAddRequest(data.DN.ValueString(), []ldap.Control{})
	a.Attribute("objectClass", objectClasses)

	ctx = MaskAttributes(ctx, attributes)

	for attributeType, values := range attributes {
		a.Attribute(attributeType, values)
	}

	tflog.Debug(ctx, "Adding LDAP entry", map[string]interface{}{
		"entry": ToLDIF(a),
	})

	return L.conn.Add(a)
}

func (L *LDAPObjectResource) isIgnored(ctx context.Context, attributeType string, data *LDAPObjectResourceModel, diagnostics diag.Diagnostics) bool {
	var ignoredAttributes []string
	diagnostics.Append(data.IgnoreChanges.ElementsAs(ctx, &ignoredAttributes, false)...)

	if diagnostics.HasError() {
		return false
	}
	return funk.ContainsString(ignoredAttributes, attributeType)
}
