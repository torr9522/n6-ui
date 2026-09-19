package templates

func Streaming() *Definition {
	return &Definition{
		Name:        "streaming",
		DisplayName: "流媒体分流",
		Description: "为常见流媒体服务生成默认域名分流规则。",
		Rules: []Rule{
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "netflix.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "nflxvideo.net"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "nflximg.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "youtube.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "googlevideo.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "youtu.be"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "disneyplus.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "disney.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "primevideo.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "aiv-cdn.net"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "hulu.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "max.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "spotify.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "twitch.tv"},
		},
	}
}
