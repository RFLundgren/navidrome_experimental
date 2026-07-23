package plugins

import (
	"context"
	"fmt"

	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/plugins/capabilities"
)

// CapabilityAction indicates the plugin can run on-demand named actions
// triggered by a user from the admin UI (e.g. a "Test Connection" button on
// the plugin's config page), as opposed to scheduled or queued execution.
// Detected when the plugin exports the nd_on_action function.
const CapabilityAction Capability = "Action"

const FuncOnAction = "nd_on_action"

func init() {
	registerCapability(
		CapabilityAction,
		FuncOnAction,
	)
}

// TriggerPluginAction runs a named action on a loaded plugin and returns its
// human-readable result. actionName should match one of the names the
// plugin declares in its manifest.json "actions" array; the plugin itself
// decides what each name does, dispatching on capabilities.ActionRequest.Name.
func (m *Manager) TriggerPluginAction(ctx context.Context, pluginID, actionName string) (string, error) {
	m.mu.RLock()
	instance, ok := m.plugins[pluginID]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("plugin %q is not loaded - enable it first", pluginID)
	}

	if !hasCapability(instance.capabilities, CapabilityAction) {
		return "", fmt.Errorf("plugin %q does not support any actions", pluginID)
	}

	log.Debug(ctx, "Triggering plugin action", "plugin", pluginID, "action", actionName)

	input := capabilities.ActionRequest{Name: actionName}
	result, err := callPluginFunction[capabilities.ActionRequest, string](ctx, instance, FuncOnAction, input)
	if err != nil {
		log.Warn(ctx, "Plugin action failed", "plugin", pluginID, "action", actionName, err)
		return "", err
	}

	log.Debug(ctx, "Plugin action completed", "plugin", pluginID, "action", actionName)
	return result, nil
}
