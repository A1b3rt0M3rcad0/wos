package commands

import (
	"reflect"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// OperationalReadSchemas inventories implemented Core contracts. Registering a
// schema does not advertise an HTTP route or MCP tool before its adapter exists.
// Historical envelopes/material retain their own numeric JSON representation;
// the new Task verification projection has its separate exact read encoding.
func OperationalReadSchemas() map[string]map[string]any {
	result := map[string]map[string]any{}
	for name, typ := range map[string]reflect.Type{
		"ObjectiveCompletionReport":           reflect.TypeFor[application.ReadResult[application.ObjectiveCompletionReport]](),
		"ObjectiveCompletionAuthorityHistory": reflect.TypeFor[application.ObjectiveCompletionAuthorityHistory](),
		"DependencyChangesPreviewQuery":       reflect.TypeFor[application.DependencyChangesPreviewQuery](),
		"DependencyChangesPreview":            reflect.TypeFor[application.DependencyChangesPreview](),
		"DependencyImpactQuery":               reflect.TypeFor[application.DependencyImpactQuery](),
		"DependencyImpactPage":                reflect.TypeFor[application.DependencyImpactPage](),
		"TaskDelivery":                        reflect.TypeFor[domain.TaskDelivery](),
		"TaskDeliveryDecision":                reflect.TypeFor[domain.TaskDeliveryDecision](),
		"TaskDeliveryWithdrawal":              reflect.TypeFor[domain.TaskDeliveryWithdrawal](),
		"CredentialAdministrativeRecord":      reflect.TypeFor[ports.CredentialAdministrativeRecord](),
		"CredentialManagementPage":            reflect.TypeFor[ports.CredentialManagementPage](),
		"CredentialRevocationImpact":          reflect.TypeFor[ports.CredentialRevocationImpact](),
		"OperationalErrorDetails":             reflect.TypeFor[domain.ErrorDetails](),
	} {
		result[name] = Schema(typ)
	}
	result["TaskVerificationView"] = WorkspaceReadSchema(reflect.TypeFor[application.TaskVerificationView]())
	return result
}
