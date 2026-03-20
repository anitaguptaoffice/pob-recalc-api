package translator

import (
	"strings"
	"testing"

	"github.com/cn-poe-community/cn-poe-utils/go/api"
)

func TestExtractIconPath(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantKey  string // substring that should appear in the decoded path
	}{
		{
			name:    "CN body armour (Silken Wrap)",
			url:     "https://poecdn.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvQXJtb3Vycy9Cb2R5QXJtb3Vycy9Cb2R5SW50MkIiLCJ3IjoyLCJoIjozLCJzY2FsZSI6MX1d/3689be0ddd/BodyInt2B.png",
			wantKey: "BodyArmours",
		},
		{
			name:    "CN helmet (The Twisting Scream)",
			url:     "https://poecdn.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvQXJtb3Vycy9IZWxtZXRzL1RoZVR3aXN0aW5nU2NyZWFtIiwidyI6MiwiaCI6Miwic2NhbGUiOjF9XQ/4815fa0fc8/TheTwistingScream.png",
			wantKey: "Helmets",
		},
		{
			name:    "CN body armour (Hussar Brigandine)",
			url:     "https://poecdn.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvQXJtb3Vycy9Cb2R5QXJtb3Vycy9Cb2R5U3RyRGV4MkIiLCJ3IjoyLCJoIjozLCJzY2FsZSI6MX1d/cd52e7b841/BodyStrDex2B.png",
			wantKey: "BodyArmours",
		},
		{
			name:    "International server URL",
			url:     "https://web.poecdn.com/image/Art/2DItems/Armours/BodyArmours/BodyInt2B.png",
			wantKey: "BodyArmours",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := extractIconPath(tt.url)
			t.Logf("Decoded path: %s", path)
			if !strings.Contains(path, tt.wantKey) {
				t.Errorf("extractIconPath(%q) = %q, want to contain %q", tt.url, path, tt.wantKey)
			}
		})
	}
}

func TestDetectSlotFromIcon(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		wantSlot string
	}{
		{
			name:     "CN body armour → Body Armour",
			url:      "https://poecdn.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvQXJtb3Vycy9Cb2R5QXJtb3Vycy9Cb2R5SW50MkIiLCJ3IjoyLCJoIjozLCJzY2FsZSI6MX1d/3689be0ddd/BodyInt2B.png",
			wantSlot: "Body Armour",
		},
		{
			name:     "CN helmet → Helmet",
			url:      "https://poecdn.game.qq.com/gen/image/WzI1LDE0LHsiZiI6IjJESXRlbXMvQXJtb3Vycy9IZWxtZXRzL1RoZVR3aXN0aW5nU2NyZWFtIiwidyI6MiwiaCI6Miwic2NhbGUiOjF9XQ/4815fa0fc8/TheTwistingScream.png",
			wantSlot: "Helmet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal item with just the icon
			item := &api.Item{Icon: tt.url}
			slot := detectSlotFromItem(item)
			if slot != tt.wantSlot {
				t.Errorf("detectSlotFromItem(icon=%q) = %q, want %q", tt.url, slot, tt.wantSlot)
			}
		})
	}
}
