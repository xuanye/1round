package game

import "testing"

func TestSettlementShareScene(t *testing.T) {
	for _, tc := range []struct {
		name  string
		token string
		want  string
	}{
		{"base64 token", "Abcdef0123456789_-Abcdef", "Abcdef0123456789_-Abcdef"},
		{"existing UUID token", "036662a6-9126-4225-bc19-ab55e1748311", "036662a691264225bc19ab55e1748311"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := settlementShareScene(tc.token)
			if err != nil || got != tc.want {
				t.Fatalf("settlementShareScene(%q) = %q, %v; want %q", tc.token, got, err, tc.want)
			}
			if len(got) > 32 {
				t.Fatalf("scene is too long: %d", len(got))
			}
		})
	}
	if _, err := settlementShareScene("123456789012345678901234567890123"); err == nil {
		t.Fatal("expected unsupported long token to be rejected before calling WeChat")
	}
}
