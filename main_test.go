package main

import "testing"

func TestResolveFilter(t *testing.T) {
	cases := []struct {
		name        string
		all         bool
		author      string
		currentMail string
		want        string
		wantErr     bool
	}{
		{"default uses current email", false, "", "me@x.com", "me@x.com", false},
		{"all clears the filter", true, "", "me@x.com", "", false},
		{"author overrides default", false, "other@y.io", "me@x.com", "other@y.io", false},
		{"all plus author conflicts", true, "other@y.io", "me@x.com", "", true},
		{"no email and no flags errors", false, "", "", "", true},
		{"author works without a configured email", false, "other@y.io", "", "other@y.io", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := resolveFilter(c.all, c.author, c.currentMail)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if got != c.want {
				t.Errorf("email = %q, want %q", got, c.want)
			}
		})
	}
}
