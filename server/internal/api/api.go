// Package api is the HTTP and websocket surface the browser talks to.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/websocket"

	"github.com/sparstrow/sparstrowgen/server/internal/auth"
	"github.com/sparstrow/sparstrowgen/server/internal/hub"
	"github.com/sparstrow/sparstrowgen/server/internal/mail"
	"github.com/sparstrow/sparstrowgen/server/internal/protocol"
	"github.com/sparstrow/sparstrowgen/server/internal/store"
)

// Config is what the deployment decides. Every field is required: there is no
// value of any of them that means "no authentication", because the one mistake
// this app cannot afford is being reachable without it.
type Config struct {
	// DaemonToken is the shared secret the owner's machine presents on a
	// DEVELOPMENT server. Empty on anything deployed, which turns that way in
	// off entirely (D-038). Kept until
	// computers are paired individually (US2).
	DaemonToken string
	// Origin is the exact browser origin allowed to call this API, e.g.
	// "https://app.sparstrow.com". Cross-origin requests from anywhere else are
	// refused, and so are websocket upgrades. Links in emails point here.
	Origin string
	// SecureCookie marks the session cookie Secure, which makes it unusable
	// over plain HTTP. On in production, off for http://localhost.
	SecureCookie bool
	// OwnerEmail is the account this deployment belongs to. It is always
	// invited, it is told about access requests, and the machine presenting
	// DaemonToken works for it.
	OwnerEmail string
	// Invited are the other addresses allowed to create an account. Approving
	// an access request means adding its address here (docs/Later.md L-18 is
	// the place in the product that replaces this).
	Invited []string
	// Mailer sends confirmation and reset links.
	Mailer mail.Sender
}

type API struct {
	store    *store.Store
	hub      *hub.Hub
	log      *slog.Logger
	cfg      Config
	throttle *auth.Throttle
	// requests slows down access requests from one client, separately from
	// sign-in attempts: a stranger flooding requests must not be able to put
	// the owner's own sign-in into back-off.
	requests *auth.Throttle
	// hashing bounds how many argon2 hashes run at once. Buffered to hashSlots;
	// a send that would block means the server is already at its limit.
	hashing chan struct{}

	// invited is OwnerEmail and Invited, normalised, for lookup.
	invited map[string]bool
	// mailGate spaces out emails to one address. See sendGap.
	mailGate *sendGate
	// sendGap is the least time between two emails of one kind to one address.
	// A field rather than a constant so a test can take it to zero.
	sendGap time.Duration

	// turns maps an in-flight turn to the conversation and entry it is writing
	// into, so a daemon message carrying only a turn id can be routed.
	mu    sync.Mutex
	turns map[string]*turn
}

type turn struct {
	// UserID is whose turn this is. It decides which browsers hear about it,
	// which machine may report on it, and who may stop it.
	UserID         string
	ConversationID string
	EntryID        string
	Provider       string
	Model          protocol.Model
	// Text accumulated from deltas, so a provider that streams and then also
	// sends a final message does not double up.
	Streamed string
}

func New(s *store.Store, h *hub.Hub, log *slog.Logger, cfg Config) *API {
	invited := map[string]bool{store.NormaliseEmail(cfg.OwnerEmail): true}
	for _, address := range cfg.Invited {
		if address = store.NormaliseEmail(address); address != "" {
			invited[address] = true
		}
	}
	return &API{
		store: s, hub: h, log: log, cfg: cfg,
		throttle: auth.NewThrottle(),
		requests: auth.NewThrottle(),
		hashing:  make(chan struct{}, hashSlots),
		invited:  invited,
		mailGate: newSendGate(),
		sendGap:  defaultSendGap,
		turns:    map[string]*turn{},
	}
}

// upgrader refuses a websocket from anywhere but the configured origin.
//
// A cookie is sent on a websocket handshake exactly as on any other request,
// and the same-origin policy does NOT apply to websockets â€” so without this
// check any page a person happened to visit could open a socket to this
// server, be authenticated by their own cookie, and read every conversation.
//
// A handshake with no Origin header at all is allowed: that is a non-browser
// client, which has no cookie to abuse and must still present the daemon token
// or a session of its own.
func (a *API) upgrader() websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			return origin == "" || origin == a.cfg.Origin
		},
	}
}

func (a *API) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(a.cors)

	// --- open to anyone -----------------------------------------------------
	// Liveness only. It deliberately does NOT report whether any machine is
	// connected: an unauthenticated endpoint should not answer "is somebody at
	// their desk right now".
	r.Get("/api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
	})
	r.Post("/api/auth/login", a.login)
	r.Post("/api/auth/logout", a.logout)
	r.Get("/api/auth/session", a.session)

	// Account access (accounts.go). Each is reachable without a session because
	// each is how somebody without one gets one; each answers only about the
	// address or link it was given.
	r.Post("/api/auth/register", a.register)
	r.Post("/api/auth/register/resend", a.resendConfirmation)
	r.Post("/api/auth/register/complete", a.completeRegistration)
	r.Post("/api/auth/links/check", a.checkLink)
	r.Post("/api/auth/password/forgot", a.forgotPassword)
	r.Post("/api/auth/password/reset", a.resetPassword)

	// The daemon presents its own token on the handshake rather than a session
	// cookie, so it is not behind requireSession.
	r.Get("/daemon", a.daemonSocket)
	r.Post("/daemon/pair", a.daemonPair)

	// --- everything else needs an account -----------------------------------
	r.Group(func(r chi.Router) {
		r.Use(a.requireSession)

		r.Get("/api/providers", a.getProviders)
		r.Get("/api/machines", a.listMachines)
		r.Post("/api/machines/pairings", a.createPairing)
		r.Get("/api/machines/pairings/{id}", a.getPairing)
		r.Post("/api/machines/pairings/{id}/approve", a.approvePairing)
		r.Post("/api/machines/pairings/{id}/decline", a.declinePairing)
		r.Get("/api/machines/{id}", a.getMachine)
		r.Delete("/api/machines/{id}", a.revokeMachine)
		r.Post("/api/machines/{id}/automatic-updates", a.setAutomaticUpdates)
		r.Post("/api/machines/{id}/updates/check", a.checkForUpdates)
		r.Post("/api/machines/{id}/updates/apply", a.applyUpdate)
		r.Get("/api/conversations", a.listConversations)
		r.Post("/api/conversations", a.createConversation)
		r.Get("/api/conversations/{id}", a.getConversation)
		r.Patch("/api/conversations/{id}", a.patchConversation)
		r.Delete("/api/conversations/{id}", a.deleteConversation)
		r.Get("/api/conversations/{id}/switch-cost", a.switchCost)
		r.Get("/api/directories", a.listDirectories)
		r.Get("/api/folders/recent", a.recentFolders)
		r.Post("/api/conversations/{id}/messages", a.postMessage)
		// Keyed by turn, not by conversation. "Stop whatever this conversation
		// is running" would be ambiguous the moment a turn ends between the
		// click and the request, and would then stop the wrong one.
		r.Post("/api/turns/{turnId}/stop", a.stopTurn)

		r.Post("/api/auth/password", a.changePassword)
		r.Get("/api/appearance", a.getAppearance)
		r.Post("/api/appearance", a.setAppearance)
		r.Get("/api/profile", a.getProfile)
		r.Post("/api/profile", a.setProfile)
		// The picture is its own resource because it is bytes, not JSON, and
		// because the browser fetches it with an <img> rather than with the
		// profile. No id in the path: it is always the signed-in account's.
		r.Get("/api/profile/avatar", a.getAvatar)
		r.Post("/api/profile/avatar", a.putAvatar)
		r.Delete("/api/profile/avatar", a.deleteAvatar)

		r.Get("/ws", a.browserSocket)
	})
	return r
}

// cors answers exactly one origin, and only with credentials allowed.
//
// "*" cannot legally be combined with credentials, and a browser enforces that
// â€” but the deeper point is that the list of origins allowed to act as a signed
// in person should be one, named in the deployment, rather than everything.
//
// Vary: Origin because the answer differs per request, and a cache that misses
// that would hand one origin's permission to another.
func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		if origin := r.Header.Get("Origin"); origin != "" && origin == a.cfg.Origin {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
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

// failConversation answers a store error about a conversation: somebody else's
// conversation and one that never existed are both a plain 404.
func (a *API) failConversation(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		a.fail(w, store.ErrNotFound, http.StatusNotFound)
		return
	}
	a.fail(w, err, http.StatusInternalServerError)
}

// ---------------------------------------------------------------------------
// conversations
// ---------------------------------------------------------------------------

func (a *API) getProviders(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	writeJSON(w, a.hub.Providers(user.ID))
}

func (a *API) listConversations(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	list, err := a.store.List(r.Context(), user.ID, r.URL.Query().Get("q"))
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (a *API) getConversation(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	c, err := a.store.Get(r.Context(), user.ID, chi.URLParam(r, "id"))
	if err != nil {
		a.failConversation(w, err)
		return
	}
	writeJSON(w, c)
}

func (a *API) createConversation(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
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
		// The folder this account last worked in beats the directory the server
		// process happens to have been started in (docs/Bugs.md B-3).
		if recent, err := a.store.RecentFolders(r.Context(), user.ID, 1); err == nil && len(recent) > 0 {
			body.Folder = recent[0]
		} else {
			body.Folder = defaultFolder()
		}
	}
	if body.Provider == "" {
		if ps := a.hub.Providers(user.ID); len(ps) > 0 && ps[0].Model != nil {
			body.Provider, body.Model = ps[0].ID, *ps[0].Model
		}
	}
	c, err := a.store.Create(r.Context(), user.ID, body.Folder, body.Provider, body.Model)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventConversation, Conversation: &c})
	writeJSON(w, c)
}

func (a *API) patchConversation(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	id := chi.URLParam(r, "id")
	var body struct {
		Title    *string `json:"title"`
		Archived *bool   `json:"archived"`
		Folder   *string `json:"folder"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		a.fail(w, err, http.StatusBadRequest)
		return
	}
	// Fetched first, for its owner, so a patch naming no field still answers
	// 404 for a conversation that is not this account's â€” rather than 200 with
	// an empty conversation.
	c, err := a.store.Get(r.Context(), user.ID, id)
	if err == nil && body.Title != nil {
		c, err = a.store.Rename(r.Context(), user.ID, id, *body.Title)
	}
	if err == nil && body.Archived != nil {
		c, err = a.store.SetArchived(r.Context(), user.ID, id, *body.Archived)
	}
	if err == nil && body.Folder != nil {
		c, err = a.store.SetFolder(r.Context(), user.ID, id, *body.Folder)
	}
	if err != nil {
		a.failConversation(w, err)
		return
	}
	c.Entries = []protocol.Entry{}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{Type: protocol.EventConversation, Conversation: &c})
	writeJSON(w, c)
}

func (a *API) deleteConversation(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	if err := a.store.Delete(r.Context(), user.ID, chi.URLParam(r, "id")); err != nil {
		a.failConversation(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// switchCost quotes what moving to a provider would cost, without spending
// anything. Selecting a provider is free; this is the number the person decides
// on, so it is computed from the real text that would be replayed.
func (a *API) switchCost(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	id := chi.URLParam(r, "id")
	// Ownership first: the replay query itself knows nothing about accounts.
	if _, err := a.store.Get(r.Context(), user.ID, id); err != nil {
		a.failConversation(w, err)
		return
	}
	unseen, err := a.store.Unseen(r.Context(), id, r.URL.Query().Get("provider"))
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
// the UI says so â€” but it is derived from the actual text that would be sent,
// which a flat per-message figure was not.
func estimateTokens(entries []protocol.Entry) int64 {
	var chars int
	for _, e := range entries {
		chars += len(e.Text)
	}
	return int64(chars / 4)
}

// ---------------------------------------------------------------------------
// sending a message â€” where a turn begins
// ---------------------------------------------------------------------------

func (a *API) postMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := userFrom(ctx)
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

	// The ownership check for everything below. The entry and provider-session
	// queries this handler goes on to use take a conversation id alone.
	conv, err := a.store.Get(ctx, user.ID, id)
	if err != nil {
		a.failConversation(w, err)
		return
	}
	if body.Provider == "" {
		body.Provider, body.Model = conv.Provider, conv.Model
	}

	// This account's machine. Another account's machine being connected is no
	// help, and must not be used.
	if !a.hub.DaemonOnline(user.ID) {
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return
	}
	// Refused before anything is written: a turn a too-old computer cannot run
	// would only fail, and the person needs to be told to update instead.
	if a.hub.DaemonTooOld(user.ID) {
		a.fail(w, errMachineTooOld, http.StatusConflict)
		return
	}

	// The replay marker is written HERE â€” at the moment the catch-up is paid
	// for â€” never when the provider was selected. Selecting is free, and the
	// transcript should say what happened, not what was contemplated.
	//
	// The condition is what this provider has not seen, and nothing else.
	// Moving a conversation to another folder drops the sessions too (they are
	// keyed to the old directory), so gating on a provider CHANGE left the next
	// turn with no history at all (docs/Bugs.md B-7).
	unseen, err := a.store.Unseen(ctx, id, body.Provider)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	var replay []protocol.ReplayEntry
	if len(unseen) > 0 {
		tokens := estimateTokens(unseen)
		marker, err := a.store.AppendReplay(ctx, id, body.Provider, body.Model, int32(len(unseen)), tokens)
		if err != nil {
			a.fail(w, err, http.StatusInternalServerError)
			return
		}
		a.hub.BroadcastTo(user.ID, protocol.ClientEvent{
			Type: protocol.EventEntryAdded, ConversationID: id, Entry: &marker,
		})
		for _, e := range unseen {
			replay = append(replay, protocol.ReplayEntry{
				Role: e.Role, Provider: e.Provider, Text: e.Text,
			})
		}
	}

	// Name it from what is being said, before the conversation is broadcast
	// below â€” so the name travels with the same event rather than needing a
	// second one. Only ever fills a blank (docs/Decisions.md D-024).
	//
	// A failure here is logged and not returned: the message is what was asked
	// for, and refusing to send it because its conversation could not be named
	// would trade the thing that matters for the thing that doesn't.
	if conv.Title == "" {
		if _, _, err := a.store.NameFrom(ctx, user.ID, id, body.Text); err != nil {
			a.log.Warn("could not name the conversation", "conversation", id, "err", err)
		}
	}

	// Broadcast it. The conversation really is on the new provider after this,
	// but without saying so the browser keeps the copy it fetched before the
	// send and the composer snaps back to the old provider (docs/Bugs.md B-9).
	switched, err := a.store.SetProvider(ctx, user.ID, id, body.Provider, body.Model)
	if err != nil {
		a.failConversation(w, err)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{
		Type: protocol.EventConversation, Conversation: &switched,
	})

	userEntry, err := a.store.AppendUser(ctx, id, body.Text)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{
		Type: protocol.EventEntryAdded, ConversationID: id, Entry: &userEntry,
	})

	agentEntry, err := a.store.AppendAgentPlaceholder(ctx, id, body.Provider, body.Model)
	if err != nil {
		a.fail(w, err, http.StatusInternalServerError)
		return
	}
	a.hub.BroadcastTo(user.ID, protocol.ClientEvent{
		Type: protocol.EventEntryAdded, ConversationID: id, Entry: &agentEntry,
	})

	turnID := agentEntry.ID
	a.mu.Lock()
	a.turns[turnID] = &turn{
		UserID: user.ID, ConversationID: id, EntryID: agentEntry.ID,
		Provider: body.Provider, Model: body.Model,
	}
	a.mu.Unlock()

	sent := a.hub.SendToDaemon(user.ID, protocol.ServerMessage{
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
		a.finishTurn(ctx, turnID, "", 0, 0, errDaemonOffline.Error(), false)
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, map[string]any{"turnId": turnID, "entry": agentEntry})
}

// stopTurn ends a turn that is already running.
//
// It does not finish the entry itself. The daemon confirms with DaemonStopped
// and that goes through the same finishTurn as every other ending, so there is
// one path for "this turn is over" rather than a second one that has to be kept
// in step with the first.
func (a *API) stopTurn(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r.Context())
	turnID := chi.URLParam(r, "turnId")

	a.mu.Lock()
	t := a.turns[turnID]
	a.mu.Unlock()
	// Another account's turn is answered exactly like one that has finished.
	// Saying "not yours" would confirm that a turn id is live.
	if t == nil || t.UserID != user.ID {
		// The click and the turn ending race by nature, so this is an ordinary
		// outcome rather than something to alarm anyone about â€” but it is not a
		// success either, because nothing was stopped.
		a.fail(w, errTurnNotRunning, http.StatusConflict)
		return
	}
	if !a.hub.SendToDaemon(user.ID, protocol.ServerMessage{
		Type: protocol.ServerStopTurn, TurnID: turnID,
	}) {
		a.fail(w, errDaemonOffline, http.StatusServiceUnavailable)
		return
	}
	// Accepted, not done: the turn ends when the daemon says it has.
	w.WriteHeader(http.StatusAccepted)
}

// ---------------------------------------------------------------------------
// daemon messages
// ---------------------------------------------------------------------------

// handleDaemonMessage acts on what an account's machine reports. A machine can
// only report on its own account's turns: a turn id belonging to anyone else is
// ignored exactly like one that does not exist.
func (a *API) handleDaemonMessage(userID string, msg protocol.DaemonMessage) {
	ctx := context.Background()

	switch msg.Type {
	case protocol.DaemonHello:
		a.hub.SetProviders(userID, msg.Providers)
		return
	case protocol.DaemonLimit:
		// Not tied to a turn's lifecycle: the window belongs to the provider,
		// and it stays true after the turn that happened to report it ends.
		a.hub.SetHeadroom(userID, msg.Provider, msg.Headroom)
		return
	}

	a.mu.Lock()
	t := a.turns[msg.TurnID]
	a.mu.Unlock()
	if t == nil || t.UserID != userID {
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
		a.hub.BroadcastTo(t.UserID, protocol.ClientEvent{
			Type: protocol.EventEntryDelta, ConversationID: t.ConversationID,
			EntryID: t.EntryID, Text: msg.Text,
		})

	case protocol.DaemonDone:
		a.finishTurn(ctx, msg.TurnID, msg.Full, msg.Tokens, msg.SpendTicks, "", false)

	case protocol.DaemonFailed:
		// Whatever arrived before it stopped is kept: a partial answer is still
		// worth reading, and deleting it would hide what went wrong.
		//
		// The daemon's text wins over the deltas we accumulated. A provider that
		// does not stream sends no deltas at all, so t.Streamed is empty and the
		// parser's recovered messages are the only copy there is (docs/Bugs.md
		// B-10).
		text := msg.Full
		if text == "" {
			text = t.Streamed
		}
		a.finishTurn(ctx, msg.TurnID, text, 0, 0, msg.Error, false)

	case protocol.DaemonStopped:
		// No failure text. The turn ended because it was asked to, and any error
		// the CLI produced while being killed is a consequence of that.
		text := msg.Full
		if text == "" {
			text = t.Streamed
		}
		a.finishTurn(ctx, msg.TurnID, text, msg.Tokens, msg.SpendTicks, "", true)
	}
}

// abandonTurns closes out every turn one account's machine was running,
// because that machine has gone.
//
// Only a message carrying a turn id ever finished a turn, and the process that
// would have sent one no longer exists â€” so without this the agent entry stays
// an empty placeholder and the composer stays locked, including after a
// refresh (docs/Bugs.md B-8). Other accounts' turns run on other machines and
// are untouched.
//
// Called only when the disconnecting socket was still the current one. A
// daemon that reconnected has already replaced it.
func (a *API) abandonTurns(userID, reason string) {
	a.mu.Lock()
	var ids []string
	for id, t := range a.turns {
		if t.UserID == userID {
			ids = append(ids, id)
		}
	}
	a.mu.Unlock()
	if len(ids) == 0 {
		return
	}

	a.log.Warn("machine went away mid-turn", "turns", len(ids))
	for _, id := range ids {
		// Text that arrived before the machine went is kept. finishTurn re-reads
		// the turn under the lock and does nothing if it has since finished.
		a.mu.Lock()
		t := a.turns[id]
		a.mu.Unlock()
		if t == nil {
			continue
		}
		a.finishTurn(context.Background(), id, t.Streamed, 0, 0, reason, false)
	}
}

func (a *API) finishTurn(ctx context.Context, turnID, text string, tokens, spendTicks int64, failure string, stopped bool) {
	a.mu.Lock()
	t := a.turns[turnID]
	delete(a.turns, turnID)
	a.mu.Unlock()
	if t == nil {
		return
	}

	// One transaction: the entry and the provider's seen-marker go together, so
	// nothing can drop the session in between and have the drop undone
	// (docs/KnownGaps.md G-18).
	entry, err := a.store.FinishTurn(ctx, store.TurnResult{
		ConversationID: t.ConversationID,
		EntryID:        t.EntryID,
		Provider:       t.Provider,
		Text:           text,
		Tokens:         tokens,
		SpendTicks:     spendTicks,
		Failure:        failure,
		Stopped:        stopped,
	})
	if err != nil {
		a.log.Error("finish entry", "err", err)
		return
	}
	a.hub.BroadcastTo(t.UserID, protocol.ClientEvent{
		Type: protocol.EventEntryDone, ConversationID: t.ConversationID, Entry: &entry,
	})

	if tokens > 0 || spendTicks > 0 {
		if conv, err := a.store.AddUsage(ctx, t.ConversationID, tokens, spendTicks); err == nil {
			a.hub.BroadcastTo(t.UserID, protocol.ClientEvent{
				Type: protocol.EventConversation, Conversation: &conv,
			})
		}
	}
}
