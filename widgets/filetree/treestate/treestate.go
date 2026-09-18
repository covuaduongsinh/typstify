// Package treestate holds TreeState, factored out of widgets/filetree so it
// can be referenced by packages that must stay UI-toolkit-agnostic (notably
// service/workspace.go, so that cmd/typstify-server -- the headless web-mode
// entrypoint, docs/plans/plan_web_version.md -- doesn't pull in gioui.org
// just to persist which folders were expanded in the desktop file tree).
// widgets/filetree.TreeState is a type alias to this, so existing code and
// serialized data are unaffected.
package treestate

// TreeState is a pure-data snapshot of a file tree's UI expand state (which
// folders are expanded).
type TreeState struct {
	Path          string
	ExpandedNodes []string
}
