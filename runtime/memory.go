package runtime

import (
	"fmt"
	"net/url"
	"strings"
)

// MemoryDecl is a node's declaration that it keeps an agent memory — a git
// repository per agent node holding what the agent learned (skills), noted
// (notes) and used (usage), keyed by flow and node rather than by robot.
//
// The Designer shows the Memory tab for ANY node that declares it, with no
// node-type special-casing; the API serves the repository; the package that
// declares it owns the worktree and the sync (through robomotion-go/memory).
// Mirrors the Python SDK's node_decorator(memory={...}).
type MemoryDecl struct {
	// Layout is the memory layout version the package writes — "v1".
	Layout string `json:"layout"`
	// Roots are the repository-relative directories the package treats as
	// content roots: "skills/learned", "skills/reviewed", "notes", "usage".
	// The worktree root itself is never one of them.
	Roots []string `json:"roots"`
	// Sync says the package pushes/pulls the repository itself (through the
	// workspace git proxy with the robot's own token). False means local
	// history only.
	Sync bool `json:"sync,omitempty"`
}

// MemoryProvider is the optional interface a node implements to declare an
// agent memory. Emitted into the pspec as `memory`, beside `assets` and
// `metadata`; a node that does not implement it produces a byte-identical
// pspec to before.
type MemoryProvider interface {
	Memory() MemoryDecl
}

// GitRemote is how a package reaches its memory repository: the smart-HTTP URL
// behind the API's git proxy, with Basic auth — the robot's id as user and its
// instance token as password, exactly as the deskbot clones flows.
type GitRemote struct {
	URL      string
	Username string
	Password string
}

// MemoryRepoName is the workspace repository an agent node's memory lives in,
// without the `wsp-<ws>/` organisation and without `.git`. One repository per
// agent node: the flow id ties the ACL to the flow's, the node guid keeps two
// agent nodes on one flow apart.
func MemoryRepoName(flowID, guid string) string {
	return "agent-" + flowID + "-" + guid
}

// MemoryRemote resolves the remote for a node's memory from the running
// robot's own access: `<ApiEndpoint>/v1/git/wsp-<workspace>/agent-<flow>-<guid>.git`
// with the instance token as password. The error is "no remote" in any
// situation where the package should fall back to local history only — CLI
// mode, a robot that has not been told its workspace, a runtime that is not
// initialised — and the caller is expected to treat it that way rather than
// fail a turn.
func MemoryRemote(flowID, guid string) (GitRemote, error) {
	if strings.TrimSpace(flowID) == "" || strings.TrimSpace(guid) == "" {
		return GitRemote{}, fmt.Errorf("no remote: a memory remote needs a flow id and a node guid")
	}
	access, err := GetInstanceAccess()
	if err != nil {
		return GitRemote{}, fmt.Errorf("no remote: %w", err)
	}
	if access == nil || strings.TrimSpace(access.ApiEndpoint) == "" || strings.TrimSpace(access.AccessToken) == "" {
		return GitRemote{}, fmt.Errorf("no remote: the instance access carries no API endpoint or token")
	}
	info, err := GetRobotInfo()
	if err != nil {
		return GitRemote{}, fmt.Errorf("no remote: %w", err)
	}
	workspace, _ := info["workspace_id"].(string)
	if strings.TrimSpace(workspace) == "" {
		return GitRemote{}, fmt.Errorf("no remote: the robot has no workspace id")
	}
	robot, _ := info["id"].(string)
	if strings.TrimSpace(robot) == "" {
		robot = "robot"
	}
	base, err := url.Parse(strings.TrimRight(strings.TrimSpace(access.ApiEndpoint), "/"))
	if err != nil || base.Host == "" {
		return GitRemote{}, fmt.Errorf("no remote: the API endpoint %q is not a URL", access.ApiEndpoint)
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + "/v1/git/wsp-" + workspace + "/" + MemoryRepoName(flowID, guid) + ".git"
	base.RawQuery, base.Fragment = "", ""
	return GitRemote{URL: base.String(), Username: robot, Password: access.AccessToken}, nil
}
