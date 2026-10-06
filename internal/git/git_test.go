package git

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want GitInfo
	}{
		{
			name: "clean with upstream",
			in: "# branch.oid abc123\n" +
				"# branch.head main\n" +
				"# branch.upstream origin/main\n" +
				"# branch.ab +0 -0\n",
			want: GitInfo{Branch: "main"},
		},
		{
			name: "ahead and behind",
			in: "# branch.head feature\n" +
				"# branch.ab +2 -3\n",
			want: GitInfo{Branch: "feature", Ahead: 2, Behind: 3},
		},
		{
			name: "no upstream (no branch.ab line)",
			in:   "# branch.head main\n",
			want: GitInfo{Branch: "main"},
		},
		{
			name: "dirty: staged, modified, untracked",
			in: "# branch.head main\n" +
				"1 M. N... 100644 100644 100644 aaa bbb staged.go\n" +
				"1 .M N... 100644 100644 100644 ccc ddd modified.go\n" +
				"1 MM N... 100644 100644 100644 eee fff both.go\n" +
				"? new1.go\n" +
				"? new2.go\n",
			want: GitInfo{Branch: "main", Staged: 2, Unstaged: 2, Untracked: 2},
		},
		{
			name: "detached head",
			in:   "# branch.head (detached)\n",
			want: GitInfo{Branch: "detached"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Parse([]byte(c.in))
			if *got != c.want {
				t.Errorf("Parse() = %+v, want %+v", *got, c.want)
			}
		})
	}
}
