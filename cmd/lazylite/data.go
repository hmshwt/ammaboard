package main

// data.go — the fixture: a deterministic snapshot of a small Go repo.
// No git subprocess, no network, no clock: the point of lazylite is the
// vocabulary, so the data only needs to be believable and stable.

import "fmt"

type fileItem struct {
	path   string
	status string // git porcelain letters: M, A, D, ??
	staged bool
	diff   []string
}

type commitItem struct {
	hash    string
	subject string
	author  string
	when    string
	body    []string
	files   []string
}

type branchItem struct {
	name    string
	current bool
	ahead   int
	behind  int
}

type repo struct {
	files    []fileItem
	commits  []commitItem
	branches []branchItem
	hashSeq  int
}

func seedRepo() *repo {
	return &repo{
		files: []fileItem{
			{path: "cmd/lazylite/main.go", status: "A", diff: []string{
				"@@ -0,0 +1,3 @@",
				"+// Command lazylite makes the tuikit vocabulary concrete.",
				"+// Screens switch. Panes tile. Cursors point.",
				"+package main",
			}},
			{path: "internal/tuikit/frame.go", status: "M", staged: true, diff: []string{
				"@@ -137,6 +137,7 @@ func (f *Frame) resolve() {",
				" 	if !f.dirty {",
				"+		// layout is stable between renders",
				" 		return",
				" 	}",
			}},
			{path: "internal/tuikit/region.go", status: "M", diff: []string{
				"@@ -16,7 +16,7 @@",
				"-	// Region is one boxed area",
				"+	// Pane is one boxed area: title, header, body",
				" type Region struct {",
			}},
			{path: "internal/tuikit/keymap.go", status: "M", diff: []string{
				"@@ -134,6 +134,9 @@",
				"+// NoShadowing checks scope discipline: no scope outside the",
				"+// global one may duplicate a global key.",
				" func NoShadowing(",
			}},
			{path: "internal/machine/cpu/execute.go", status: "M", diff: []string{
				"@@ -16,7 +16,7 @@ func (c *CPU) execute(",
				"-	fn, ok := lookup(in.Op)",
				"+	fn, ok := lookup(in.Op) // registry routes by opcode",
			}},
			{path: "docs/tuikit.md", status: "M", staged: true, diff: []string{
				"@@ -40,2 +40,6 @@",
				"+## The naming table",
				"+",
				"+Screens switch. Panes tile. Cursors point. Details stack.",
			}},
			{path: "docs/critique.md", status: "A", diff: []string{
				"@@ -0,0 +1,4 @@",
				"+# AmmaBoard — complete critique compendium",
				"+",
				"+Pre-pivot, post-pivot, further steps.",
			}},
			{path: "go.mod", status: "M", diff: []string{
				"@@ -3,3 +3,3 @@",
				"-	go 1.24.0",
				"+	go 1.25.0",
			}},
			{path: "README.md", status: "??", diff: []string{"(untracked file)"}},
			{path: "LICENSE", status: "??", diff: []string{"(untracked file)"}},
		},
		commits: []commitItem{
			{hash: "e7a11c2", subject: "wip: cursor architecture", author: "you", when: "2h ago",
				body: []string{
					"The app always knows what the cursor points at.",
					"esc, ? and : answer for wherever that is.",
				},
				files: []string{"internal/tuikit/cursor.go", "internal/tuikit/app.go"}},
			{hash: "0e7ada8", subject: "tuikit new architecture", author: "you", when: "5w ago",
				body: []string{
					"Screens, grid, panes, receipts.",
					"Extraction-ready: zero ammaboard dependencies.",
				},
				files: []string{"internal/tuikit/app.go", "internal/tuikit/screen.go"}},
			{hash: "71a56c0", subject: "tuikit improvement", author: "you", when: "5w ago",
				body:  []string{"The resize cascade; region constraints as data."},
				files: []string{"internal/tuikit/frame.go"}},
			{hash: "9f0cc50", subject: "redesign bit pattern 7+3+3+3+16", author: "you", when: "5w ago",
				body:  []string{"Opcode 7 bits, immediate 16 bits; headroom for stages 3-5."},
				files: []string{"internal/isa/isa.go"}},
			{hash: "bfd0e77", subject: "add tui", author: "you", when: "5w ago",
				body:  []string{"Three panes over the live machine."},
				files: []string{"internal/tui/board.go"}},
			{hash: "1900be2", subject: "docs rewrite", author: "you", when: "5w ago",
				body:  []string{"Architecture-first docs."},
				files: []string{"docs/architecture.md"}},
			{hash: "c21fb52", subject: "wip: tuikit first implementation", author: "you", when: "5w ago",
				body:  []string{"Frame, regions, keymaps as data."},
				files: []string{"internal/tuikit/"}},
			{hash: "490549e", subject: "feat: implement poc", author: "you", when: "5w ago",
				body:  []string{"The motherboard broker: phases, posted writes, bus."},
				files: []string{"internal/machine/"}},
		},
		branches: []branchItem{
			{name: "main", current: true, ahead: 2, behind: 0},
			{name: "feature/cursor", current: false, ahead: 5, behind: 1},
			{name: "fix/pad-width", current: false, ahead: 1, behind: 3},
			{name: "experiment/palette", current: false, ahead: 9, behind: 0},
		},
	}
}

func (r *repo) removeFile(path string) {
	for i := range r.files {
		if r.files[i].path == path {
			r.files = append(r.files[:i], r.files[i+1:]...)
			return
		}
	}
}

func (r *repo) removeBranch(name string) {
	for i := range r.branches {
		if r.branches[i].name == name {
			r.branches = append(r.branches[:i], r.branches[i+1:]...)
			return
		}
	}
}

func (r *repo) fakeHash() string {
	r.hashSeq++
	return fmt.Sprintf("%07x", 0xc0ffee0+r.hashSeq)
}

func (r *repo) stagedCount() int {
	n := 0
	for i := range r.files {
		if r.files[i].staged {
			n++
		}
	}
	return n
}
