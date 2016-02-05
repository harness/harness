package model

type System struct {
	Version   string   `json:"version"`
	Link      string   `json:"link_url"`
	Plugins   []string `json:"plugins"`
	Globals   []string `json:"globals"`
	Escalates []string `json:"privileged_plugins"`
}
