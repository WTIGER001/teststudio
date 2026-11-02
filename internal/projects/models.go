package projects

type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Archived    bool   `json:"archived"`
}

type Application struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Environment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Application string            `json:"applicationId"`
	Stage       string            `json:"stage"`
	Protected   bool              `json:"protected"`
	Disabled    bool              `json:"disabled"`
	Endpoints   map[string]string `json:"endpoints"`
	Credentials map[string]string `json:"credentialBindings,omitempty"`
	Locations   []string          `json:"executionLocations,omitempty"`
}

type Suite struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Applications   []string `json:"applications"`
	RequiresTarget bool     `json:"requiresTarget"`
	ActiveRelease  string   `json:"activeReleaseId,omitempty"`
	Archived       bool     `json:"archived"`
}
