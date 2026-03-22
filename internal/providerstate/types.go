package providerstate

type ModelCost struct {
	Input float64 `json:"input"`
}

type ModelInfo struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Status string    `json:"status,omitempty"`
	Cost   ModelCost `json:"cost,omitempty"`
}

type ProviderInfo struct {
	ID     string               `json:"id"`
	Name   string               `json:"name"`
	Env    []string             `json:"env,omitempty"`
	API    string               `json:"api,omitempty"`
	NPM    string               `json:"npm,omitempty"`
	Models map[string]ModelInfo `json:"models"`
}

type Source string

const (
	SourceEnv    Source = "env"
	SourceConfig Source = "config"
	SourceCustom Source = "custom"
	SourceAPI    Source = "api"
)

type ActiveProvider struct {
	ID      string
	Name    string
	Source  Source
	APIKey  string
	BaseURL string
	Type    string
	Models  []string
}

type State struct {
	Providers map[string]ActiveProvider
	Default   string // provider/model
	Small     string // provider/model
}

