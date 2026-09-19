package templates

func Game() *Definition {
	return &Definition{
		Name:        "game",
		DisplayName: "游戏分流",
		Description: "为常见游戏平台生成默认域名分流规则。",
		Rules: []Rule{
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "steampowered.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "steamcontent.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "steamstatic.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "steamcommunity.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "riotgames.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "leagueoflegends.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "pvp.net"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "epicgames.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "epicgamescdn.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "battle.net"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "blizzard.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "playstation.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "xbox.com"},
			{RuleType: "domain", MatchMode: "suffix", MatchValue: "nintendo.com"},
		},
	}
}
