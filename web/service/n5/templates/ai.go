package templates

func AI() *Definition {
	return &Definition{
		Name:        "ai",
		DisplayName: "AI 分流",
		Description: "为常见 AI 服务生成默认域名分流规则。",
		Rules: []Rule{
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "openai.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "chatgpt.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "oaistatic.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "oaiusercontent.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "anthropic.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "claude.ai"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "claude.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "perplexity.ai"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "githubcopilot.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "copilot.microsoft.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "gemini.google.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "grok.com"},
		},
	}
}
