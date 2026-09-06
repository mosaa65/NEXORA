package db

import "testing"

func TestSmartHubRuleKeyCanonicalizesRuleValues(t *testing.T) {
	first := smartHubRuleKey("series", HubRule{
		Types:  []string{"series"},
		TagsAny: []string{"تركي", "دراما"},
	})
	second := smartHubRuleKey("series", HubRule{
		Types:  []string{"series"},
		TagsAny: []string{"دراما", "تركي", "تركي"},
	})

	if first != second {
		t.Fatalf("expected equivalent hub rules to share a key: %q != %q", first, second)
	}
}
