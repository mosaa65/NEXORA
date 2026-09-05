package db

import "testing"

func TestMediaShowcaseAccentUsesMediaMetadata(t *testing.T) {
	tests := []struct {
		name   string
		kind   string
		genres []string
		want   string
	}{
		{name: "disney", kind: "movie", genres: []string{"ديزني", "عائلي"}, want: "cyan"},
		{name: "anime", kind: "series", genres: []string{"أنمي", "ياباني"}, want: "rose"},
		{name: "turkish", kind: "series", genres: []string{"تركي", "دراما"}, want: "amber"},
		{name: "arabic", kind: "series", genres: []string{"عربي"}, want: "emerald"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mediaShowcaseAccent(test.kind, test.genres); got != test.want {
				t.Fatalf("mediaShowcaseAccent() = %q, want %q", got, test.want)
			}
		})
	}
}
