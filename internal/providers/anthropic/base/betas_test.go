package base

import (
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMergeBetaHeader(t *testing.T) {
	cases := []struct {
		name     string
		existing string
		add      []string
		want     []string
	}{
		{"empty existing", "", []string{"a", "b"}, []string{"a", "b"}},
		{"existing only", "x", nil, []string{"x"}},
		{"dedup", "a, b", []string{"b", "c"}, []string{"a", "b", "c"}},
		{"trim spaces", " a , b ", []string{"c"}, []string{"a", "b", "c"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := MergeBetaHeader(tc.existing, tc.add)
			gotParts := strings.Split(got, ",")
			for i := range gotParts {
				gotParts[i] = strings.TrimSpace(gotParts[i])
			}
			sort.Strings(gotParts)
			want := append([]string(nil), tc.want...)
			sort.Strings(want)
			require.Equal(t, want, gotParts)
		})
	}
}
