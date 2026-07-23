package capabilities

// Action provides on-demand plugin actions triggered by a user from the
// admin UI (e.g. a "Test Connection" button on the plugin's config page),
// as opposed to scheduled or queued execution. Plugins declare which named
// actions they support via manifest.json's "actions" array; the host only
// calls this capability for actions the plugin has declared there.
//
//nd:capability name=action
type Action interface {
	// OnAction is called when a user triggers a named action from the
	// plugin's config page. The returned string is a short, human-readable
	// result shown directly to the user in the UI (not logged) - e.g.
	// "OK - test classification succeeded". Return an error for expected
	// failures (e.g. an invalid API key); the error message is what gets
	// displayed, so make it actionable.
	//nd:export name=nd_on_action
	OnAction(ActionRequest) (string, error)
}

// ActionRequest identifies which declared action the user triggered.
type ActionRequest struct {
	// Name is the action's name, matching one declared in manifest.json's
	// "actions" array.
	Name string `json:"name"`
}
