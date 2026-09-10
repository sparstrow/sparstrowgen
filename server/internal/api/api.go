// Package api is the HTTP and websocket surface the browser talks to.
package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

type API struct {
	store *store.Store
	hub   *hub.Hub
	log   *slog.Logger

	// turns maps an in-flight turn to the conversation and entry it is writing
	// into, so a daemon message carrying only a turn id can be routed.
	mu    sync.Mutex
	turns map[string]*turn
}

type turn struct {
	ConversationID string
	EntryID        string
	Provider       string
	Model          protocol.Model
	// Text accumulated from deltas, so a provider that streams and then also
	// sends a final message does not double up.
	Streamed string
}

func New(s *store.Store, h *hub.Hub, log *slog.Logger) *API {
	a := &API{store: s, hub: h, log: log, turns: map[string]*turn{}}
	h.OnDaemonMessage = a.handleDaemonMessage
	return a
}

// upgrader accepts any origin: the server is reached over the loopback address
// in development and behind our own auth in production, and there is no
// cookie-authenticated state for a foreign page to abuse yet. Revisit the day
// there is a session cookie.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func (a *API) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(cors)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "daemon": a.hub.DaemonOnline()})
	})
	r.Get("/api/providers", a.getProviders)
	r.Get("/api/conversations", a.listConversations)
	r.Post("/api/conversations", a.createConversation)
	r.Get("/api/conversations/{id}", a.getConversation)
	r.Patch("/api/conversations/{id}", a.patchConversation)
	r.Delete("/api/conversations/{id}", a.deleteConversation)
	r.Get("/api/conversations/{id}/switch-cost", a.switchCost)
	r.Post("/api/conversations/{id}/messages", a.postMessage)

	r.Get("/ws", a.browserSocket)
	r.Get("/daemon", a.daemonSocket)
	return r
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (a *API) fail(w http.ResponseWriter, err error, code int) {
	a.log.Error("request failed", "err", err)
	w.WriteHeader(code)
	writeJSON(w, map[string]string{"error": err.Error()})
}

// ---------------------------------------------------------------------------
// conversations
// ---------------------------------------------------------------------------

func (a *API) getProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, a.hub.Providers())
}

func (a *API) listConversations(w http.ResponseWriter, r *http.Request) {
	list, err := a.store.List(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (a *API) getConversation(w http.ResponseWriter, r *http.Request) {
	c, err := a.store.Get(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		a.fail(w, err, http.StatusNotFound)
		return
	}
	writeJSON(w, c)
}

func (a *API) createConversation(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Folder   string         `json:"folder"`
		Provider string         `json:"provider"`
		Model    protocol.Model `json:"model"`
	}
	// An empty body is a legitimate "new conversation with the defaults", but a
	// malformed one is not: ignoring the error here silently fell back to the
	// defaults and looked like the request had been honoured.
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	if body.Folder == "" {
		body.Folder = defaultFolder()
	}
	if body.Provider == "" {
		if ps := a.hub.Providers(); len(ps) > 0 && ps[0].Model != nil {
			body.Provider, body.Model = ps[0].ID, *ps[0].Model
		}
	}
	c, err := a.store.Create(r.Context(), body.Folder, body.Provider, body.Model)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.hub.Broadcast(protocol.ClientEvent{Type: protocol.EventConversation, Conversation: &c})
	writeJSON(w, c)
}

func (a *API) patchConversation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Title    *string `json:"title"`
		Archived *bool   `json:"archived"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	var (
		c   protocol.Conversation
		err error
	)
	if body.Title != nil {
		c, err = a.store.Rename(r.Context(), id, *body.Title)
	}
	if err == nil && body.Archived != nil {
		c, err = a.store.SetArchived(r.Context(), id, *body.Archived)
	}
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.hub.Broadcast(protocol.ClientEvent{Type: protocol.EventConversation, Conversation: &c})
	writeJSON(w, c)
}

func (a *API) deleteConversation(w http.ResponseWriter, r *http.Request) {
	if err := a.store.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// switchCost quotes what moving to a provider would cost, without spending
// anything. Selecting a provider is free; this is the number the owner decides
// on, so it is computed from the real text that would be replayed.
func (a *API) switchCost(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	unseen, err := a.store.Unseen(r.Context(), chi.URLParam(r, "id"), provider)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{
		"messagesToReplay": len(unseen),
		"estimatedTokens":  estimateTokens(unseen),
	})
}

// estimateTokens is chars/4, the standard rough ratio. It is an estimate and
// the UI says so — but it is derived from the actual text that would be sent,
// which a flat per-message figure was not.
func estimateTokens(entries []protocol.Entry) int64 {
	var chars int
	for _, e := range entries {
		chars += len(e.Text)
	}
	return int64(chars / 4)
}

// ---------------------------------------------------------------------------
// sending a message — where a turn begins
// ---------------------------------------------------------------------------

func (a *API) postMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := chi.URLParam(r, "id")

	var body struct {
		Text     string         `json:"text"`
		Provider string         `json:"provider"`
		Model    protocol.Model `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}

	conv, err := a.store.Get(ctx, id)
	if err != nil {
		a.fail(w, err, http.StatusNotFound)
		return
	}
	if body.Provider == "" {
		body.Provider, body.Model = conv.Provider, conv.Model
	}

	if !a.hub.DaemonOnline() {
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return
	}

	// The replay marker is written HERE — at the moment the catch-up is paid
	// for — never when the provider was selected. Selecting is free, and the
	// transcript should say what happened, not what was contemplated.
	unseen, err := a.store.Unseen(ctx, id, body.Provider)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	var replay []protocol.ReplayEntry
	if body.Provider != conv.Provider && len(unseen) > 0 {
		tokens := estimateTokens(unseen)
		marker, err := a.store.AppendReplay(ctx, id, body.Provider, body.Model, int32(len(unseen)), tokens)
		if err != nil {
			a.fail(w, err, http.StatusInternalServerError)
			return
		}
		a.hub.Broadcast(protocol.ClientEvent{
			Type: protocol.EventEntryAdded, ConversationID: id, Entry: &marker,
		})
		for _, e := range unseen {
			replay = append(replay, protocol.ReplayEntry{
				Role: e.Role, Provider: e.Provider, Text: e.Text,
			})
		}
	}

	if err := a.store.SetProvider(ctx, id, body.Provider, body.Model); err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}

	userEntry, err := a.store.AppendUser(ctx, id, body.Text)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.hub.Broadcast(protocol.ClientEvent{
		Type: protocol.EventEntryAdded, ConversationID: id, Entry: &userEntry,
	})

	agentEntry, err := a.store.AppendAgentPlaceholder(ctx, id, body.Provider, body.Model)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.hub.Broadcast(protocol.ClientEvent{
		Type: protocol.EventEntryAdded, ConversationID: id, Entry: &agentEntry,
	})

	turnID := agentEntry.ID
	a.mu.Lock()
	a.turns[turnID] = &turn{
		ConversationID: id, EntryID: agentEntry.ID,
		Provider: body.Provider, Model: body.Model,
	}
	a.mu.Unlock()

	sent := a.hub.SendToDaemon(protocol.ServerMessage{
		Type: protocol.ServerRunTurn,
		Turn: &protocol.RunTurn{
			TurnID:          turnID,
			ConversationID:  id,
			EntryID:         agentEntry.ID,
			Provider:        body.Provider,
			Model:           body.Model,
			Cwd:             conv.Folder,
			Prompt:          body.Text,
			ResumeSessionID: a.store.ResumeID(ctx, id, body.Provider),
			Replay:          replay,
		},
	})
	if !sent {
		a.finishTurn(ctx, turnID, "", 0, 0, errDaemonOffline.Error())
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, map[string]any{"turnId": turnID, "entry": agentEntry})
}

// ---------------------------------------------------------------------------
// daemon messages
// ---------------------------------------------------------------------------

func (a *API) handleDaemonMessage(msg protocol.DaemonMessage) {
	ctx := context.Background()

	switch msg.Type {
	case protocol.DaemonHello:
		a.hub.SetProviders(msg.Providers)
		return
	case protocol.DaemonLimit:
		// Not tied to a turn's lifecycle: the window belongs to the provider,
		// and it stays true after the turn that happened to report it ends.
		a.hub.SetHeadroom(msg.Provider, msg.Headroom)
		return
	}

	a.mu.Lock()
	t := a.turns[msg.TurnID]
	a.mu.Unlock()
	if t == nil {
		return
	}

	switch msg.Type {
	case protocol.DaemonStarted:
		// Pin the resume pointer immediately. If the turn then dies, the
		// provider session is still reusable rather than orphaned.
		if msg.SessionID != "" {
			seq, _ := a.store.LastSeq(ctx, t.ConversationID)
			_ = a.store.MarkSeen(ctx, t.ConversationID, t.Provider, msg.SessionID, seq)
		}

	case protocol.DaemonDelta:
		t.Streamed += msg.Text
		if err := a.store.AppendDelta(ctx, t.EntryID, msg.Text); err != nil {
			a.log.Error("append delta", "err", err)
		}
		a.hub.Broadcast(protocol.ClientEvent{
			Type: protocol.EventEntryDelta, ConversationID: t.ConversationID,
			EntryID: t.EntryID, Text: msg.Text,
		})

	case protocol.DaemonDone:
		a.finishTurn(ctx, msg.TurnID, msg.Full, msg.Tokens, msg.SpendTicks, "")

	case protocol.DaemonFailed:
		// Whatever arrived before it stopped is kept: a partial answer is still
		// worth reading, and deleting it would hide what went wrong.
		a.finishTurn(ctx, msg.TurnID, t.Streamed, 0, 0, msg.Error)
	}
}

func (a *API) finishTurn(ctx context.Context, turnID, text string, tokens, spendTicks int64, failure string) {
	a.mu.Lock()
	t := a.turns[turnID]
	delete(a.turns, turnID)
	a.mu.Unlock()
	if t == nil {
		return
	}

	entry, err := a.store.FinishAgent(ctx, t.EntryID, text, tokens, spendTicks, failure)
	if err != nil {
		a.log.Error("finish entry", "err", err)
		return
	}
	a.hub.Broadcast(protocol.ClientEvent{
		Type: protocol.EventEntryDone, ConversationID: t.ConversationID, Entry: &entry,
	})

	if tokens > 0 || spendTicks > 0 {
		if conv, err := a.store.AddUsage(ctx, t.ConversationID, tokens, spendTicks); err == nil {
			a.hub.Broadcast(protocol.ClientEvent{
				Type: protocol.EventConversation, Conversation: &conv,
			})
		}
	}

	// The provider has now seen everything up to and including its own reply,
	// so a switch away and back replays only what comes after this point.
	if seq, err := a.store.LastSeq(ctx, t.ConversationID); err == nil {
		_ = a.store.MarkSeen(ctx, t.ConversationID, t.Provider, "", seq)
	}
}
