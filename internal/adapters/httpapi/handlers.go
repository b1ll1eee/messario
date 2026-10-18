package httpapi

import (
	"net/http"

	"github.com/b1ll1eee/messario/internal/app"
	"github.com/b1ll1eee/messario/internal/domain"
)

// ---- auth ----

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	u, token, err := s.app.Login(r.Context(), in.Username, in.Password, r.UserAgent())
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	s.setSessionCookie(w, token, s.cfg.SessionTTL)
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(s.cfg.CookieName); err == nil {
		_ = s.app.Logout(r.Context(), c.Value)
	}
	s.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, app.UserFrom(r.Context()))
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Current string `json:"current"`
		Next    string `json:"next"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	if err := s.app.ChangePassword(r.Context(), app.UserFrom(r.Context()), in.Current, in.Next); err != nil {
		s.writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) directory(w http.ResponseWriter, r *http.Request) {
	users, err := s.app.Directory(r.Context(), app.UserFrom(r.Context()), r.URL.Query().Get("q"), int(queryInt(r, "limit")))
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// ---- chats ----

func (s *Server) listChats(w http.ResponseWriter, r *http.Request) {
	chats, err := s.app.ListChats(r.Context(), app.UserFrom(r.Context()))
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, chats)
}

func (s *Server) createChat(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Type      string  `json:"type"`
		Title     string  `json:"title"`
		MemberIDs []int64 `json:"member_ids"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	c, err := s.app.CreateGroup(r.Context(), app.UserFrom(r.Context()), domain.ChatType(in.Type), in.Title, in.MemberIDs)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) openDirect(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID int64 `json:"user_id"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	c, err := s.app.OpenDirect(r.Context(), app.UserFrom(r.Context()), in.UserID)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) getChat(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	c, m, err := s.app.GetChat(r.Context(), app.UserFrom(r.Context()), id)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"chat": c, "me": m})
}

func (s *Server) renameChat(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	var in struct {
		Title string `json:"title"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	if err := s.app.RenameChat(r.Context(), app.UserFrom(r.Context()), id, in.Title); err != nil {
		s.writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	msgs, err := s.app.History(r.Context(), app.UserFrom(r.Context()), id, queryInt(r, "after_seq"), queryInt(r, "before_seq"), int(queryInt(r, "limit")))
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, msgs)
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	var in struct {
		ClientMsgID string `json:"client_msg_id"`
		Body        string `json:"body"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	m, created, err := s.app.Send(r.Context(), app.UserFrom(r.Context()), id, in.ClientMsgID, in.Body)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(w, status, m)
}

func (s *Server) markRead(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	var in struct {
		Seq int64 `json:"seq"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	if err := s.app.MarkRead(r.Context(), app.UserFrom(r.Context()), id, in.Seq); err != nil {
		s.writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) readInfo(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	info, err := s.app.ReadInfo(r.Context(), app.UserFrom(r.Context()), id, queryInt(r, "seq"), int(queryInt(r, "limit")))
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) members(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	ms, err := s.app.Members(r.Context(), app.UserFrom(r.Context()), id)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ms)
}

func (s *Server) addMembers(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	var in struct {
		UserIDs []int64 `json:"user_ids"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	if err := s.app.AddMembers(r.Context(), app.UserFrom(r.Context()), id, in.UserIDs); err != nil {
		s.writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	uid, err := pathInt(r, "uid")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	if err := s.app.RemoveMember(r.Context(), app.UserFrom(r.Context()), id, uid); err != nil {
		s.writeAppError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- admin ----

func (s *Server) adminCreateUser(w http.ResponseWriter, r *http.Request) {
	var in app.NewUserInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	u, err := s.app.CreateUser(r.Context(), app.UserFrom(r.Context()), in)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) adminUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	var in app.UpdateUserInput
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	u, err := s.app.UpdateUser(r.Context(), app.UserFrom(r.Context()), id, in)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) adminCreateBot(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		s.writeAppError(w, err)
		return
	}
	b, token, err := s.app.CreateBot(r.Context(), app.UserFrom(r.Context()), in.Username, in.DisplayName)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"bot": b, "token": token})
}

func (s *Server) adminBotToken(w http.ResponseWriter, r *http.Request) {
	id, err := pathInt(r, "id")
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	token, err := s.app.RegenerateBotToken(r.Context(), app.UserFrom(r.Context()), id)
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

func (s *Server) adminAudit(w http.ResponseWriter, r *http.Request) {
	entries, err := s.app.AuditLog(r.Context(), app.UserFrom(r.Context()), int(queryInt(r, "limit")))
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) adminLicense(w http.ResponseWriter, r *http.Request) {
	info, err := s.app.LicenseInfo(r.Context(), app.UserFrom(r.Context()))
	if err != nil {
		s.writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}
