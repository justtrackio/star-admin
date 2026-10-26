package internal

type FrontendConfigResponse struct {
	Columns  []string            `json:"columns"`
	Settings []map[string]string `json:"settings"`
}
