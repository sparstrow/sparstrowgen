package api

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

/* Workspaces: separate areas of work inside one account (docs/Decisions.md
D-050). The owner's words for what they are for: "I would create one personal
and one work related workspace. Keeping both of them Separate."

So the test that matters most is the last one here — two workspaces, and
neither one's conversations, searches or folders showing up in the other. */

func (r *rig) workspaces() []protocol.Workspace {
	r.t.Helper()
	res := r.get("/api/workspaces")
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("list workspaces: %s", res.Status)
	}
	var out []protocol.Workspace
	decodeInto(r.t, res, &out)
	return out
}

func (r *rig) newWorkspace(name string) protocol.Workspace {
	r.t.Helper()
	res := r.post("/api/workspaces", map[string]any{"name": name})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("create workspace %q: %s", name, res.Status)
	}
	var w protocol.Workspace
	decodeInto(r.t, res, &w)
	return w
}

// conversationIn starts one through the API, in a named workspace.
func (r *rig) conversationIn(workspaceID, folder string) protocol.Conversation {
	r.t.Helper()
	res := r.post("/api/conversations", map[string]any{
		"workspaceId": workspaceID, "folder": folder,
		"provider": "claude", "model": protocol.Model{ID: "m1", Label: "M One"},
	})
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("create conversation in %s: %s", workspaceID, res.Status)
	}
	var c protocol.Conversation
	decodeInto(r.t, res, &c)
	return c
}

func (r *rig) listIn(workspaceID, query string) []protocol.Conversation {
	r.t.Helper()
	path := "/api/conversations?workspace=" + workspaceID
	if query != "" {
		path += "&q=" + query
	}
	res := r.get(path)
	if res.StatusCode != http.StatusOK {
		r.t.Fatalf("list %s: %s", path, res.Status)
	}
	var out []protocol.Conversation
	decodeInto(r.t, res, &out)
	return out
}

func TestAWorkspaceIsCreatedAndListed(t *testing.T) {
	r := newRig(t)

	// The rig's account already has the one its claim() made, the way a
	// migrated account does. Adding the second is the owner's actual scenario.
	work := r.newWorkspace("Work")
	if work.Role != store.RoleOwner {
		t.Errorf("the person who made it has role %q, want owner", work.Role)
	}

	got := r.workspaces()
	if len(got) != 2 {
		t.Fatalf("%d workspaces, want 2: %+v", len(got), got)
	}
	// Ordered by name, because a list of two or three is read to find one.
	if got[0].Name != "Personal" || got[1].Name != "Work" {
		t.Errorf("order = %q then %q, want Personal then Work", got[0].Name, got[1].Name)
	}
}

func TestAWorkspaceIsRenamedAndTheNameIsTidied(t *testing.T) {
	r := newRig(t)
	w := r.newWorkspace("  Client work  ")
	if w.Name != "Client work" {
		t.Errorf("name = %q, want it trimmed", w.Name)
	}

	res := r.do(http.MethodPatch, "/api/workspaces/"+w.ID, map[string]any{"name": "Contoso"})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("rename: %s", res.Status)
	}
	var renamed protocol.Workspace
	decodeInto(t, res, &renamed)
	if renamed.ID != w.ID || renamed.Name != "Contoso" {
		t.Errorf("renamed = %+v, want the same workspace called Contoso", renamed)
	}
}

func TestAWorkspaceNeedsAName(t *testing.T) {
	r := newRig(t)
	before := len(r.workspaces())

	for _, c := range []struct {
		what, name string
	}{
		{"empty", ""},
		{"only spaces", "   "},
		{"too long", strings.Repeat("x", store.MaxWorkspaceName+1)},
	} {
		if res := r.post("/api/workspaces", map[string]any{"name": c.name}); res.StatusCode != http.StatusBadRequest {
			t.Errorf("a %s name = %s, want 400", c.what, res.Status)
		}
	}
	// A limit counted in bytes would cut a name in a non-Latin script shorter
	// than the same-length one in ASCII — the same rule as the display name.
	long := strings.Repeat("é", store.MaxWorkspaceName)
	if res := r.post("/api/workspaces", map[string]any{"name": long}); res.StatusCode != http.StatusOK {
		t.Errorf("a name of %d characters = %s, want it accepted", store.MaxWorkspaceName, res.Status)
	}
	if n := len(r.workspaces()); n != before+1 {
		t.Errorf("%d workspaces after the refusals, want %d — only the valid one", n, before+1)
	}
}

func TestANamelessWorkspaceIsNotAWorkspace(t *testing.T) {
	r := newRig(t)
	// Every route that acts on one names it. Nothing falls back to "the first
	// workspace we find", because that is how a conversation ends up somewhere
	// the person was not looking.
	for _, path := range []string{"/api/conversations", "/api/folders/recent"} {
		if res := r.get(path); res.StatusCode != http.StatusBadRequest {
			t.Errorf("GET %s with no workspace = %s, want 400", path, res.Status)
		}
	}
	if res := r.post("/api/conversations", map[string]any{}); res.StatusCode != http.StatusBadRequest {
		t.Errorf("creating a conversation with no workspace = %s, want 400", res.Status)
	}
	// A workspace id that is not even a uuid is not one of this account's, the
	// same as a well-formed id belonging to nobody.
	for _, id := range []string{"not-a-uuid", "00000000-0000-0000-0000-000000000000"} {
		if res := r.get("/api/conversations?workspace=" + id); res.StatusCode != http.StatusNotFound {
			t.Errorf("GET a conversation list for %q = %s, want 404", id, res.Status)
		}
	}
}

// The feature, in one test: one account, two workspaces, and nothing from
// either turning up in the other.
func TestTwoWorkspacesKeepTheirWorkApart(t *testing.T) {
	r := newRig(t)
	personal := r.workspaces()[0]
	work := r.newWorkspace("Work")

	mine := r.conversationIn(personal.ID, "D:\\home\\garden-notes")
	theirs := r.conversationIn(work.ID, "D:\\clients\\contoso")
	if _, err := r.store.AppendUser(t.Context(), mine.ID, "renew the seed order"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.store.AppendUser(t.Context(), theirs.ID, "the contoso EDI mapping"); err != nil {
		t.Fatal(err)
	}

	// Each conversation says which workspace it is in, because events are
	// addressed to an account rather than to a workspace and the browser has
	// to be able to tell.
	if mine.WorkspaceID != personal.ID || theirs.WorkspaceID != work.ID {
		t.Errorf("conversations report workspaces %q and %q, want %q and %q",
			mine.WorkspaceID, theirs.WorkspaceID, personal.ID, work.ID)
	}

	for _, c := range []struct {
		what, workspace, want string
	}{
		{"Personal", personal.ID, mine.ID},
		{"Work", work.ID, theirs.ID},
	} {
		list := r.listIn(c.workspace, "")
		if len(list) != 1 || list[0].ID != c.want {
			t.Errorf("%s holds %d conversations, want only its own", c.what, len(list))
		}
	}

	// Search does not reach across either. A search is the likeliest way for
	// one area of work to leak into the other, because it is the one thing
	// that deliberately looks inside transcripts.
	if hits := r.listIn(personal.ID, "contoso"); len(hits) != 0 {
		t.Errorf("searching Personal for a word only in Work found %d", len(hits))
	}
	if hits := r.listIn(work.ID, "seed+order"); len(hits) != 0 {
		t.Errorf("searching Work for a word only in Personal found %d", len(hits))
	}

	// Nor do the folders, which say as much about what somebody is doing as
	// the conversations do.
	var folders map[string][]string
	decodeInto(t, r.get("/api/folders/recent?workspace="+personal.ID), &folders)
	if len(folders["folders"]) != 1 || folders["folders"][0] != "D:\\home\\garden-notes" {
		t.Errorf("Personal's recent folders = %v, want only its own", folders["folders"])
	}
}
