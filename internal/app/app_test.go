package app

import "testing"

func TestShouldShowActivity(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want bool
	}{
		{name: "push", args: []string{"push"}, want: true},
		{name: "status json", args: []string{"status", "--json"}, want: false},
		{name: "help", args: []string{"help"}, want: false},
		{name: "version", args: []string{"version"}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldShowActivity(tc.args); got != tc.want {
				t.Fatalf("shouldShowActivity(%v) = %v, want %v", tc.args, got, tc.want)
			}
		})
	}
}
