package templates

type Rule struct {
	RuleType   string `json:"ruleType"`
	MatchMode  string `json:"matchMode"`
	MatchValue string `json:"matchValue"`
}

type Definition struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Rules       []Rule `json:"rules"`
}

func All() []*Definition {
	return []*Definition{
		AI(),
		Game(),
		Streaming(),
	}
}

func Find(name string) *Definition {
	for _, item := range All() {
		if item.Name == name {
			return item
		}
	}
	return nil
}
