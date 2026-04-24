package analytics

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type Server struct {
	store    *Store
	telegram *Telegram
	token    string
	mux      *http.ServeMux
}

func NewServer(store *Store, telegram *Telegram, token string) *Server {
	s := &Server{
		store:    store,
		telegram: telegram,
		token:    token,
		mux:      http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("/track", s.handleTrack)
	s.mux.HandleFunc("/install", s.handleInstall)
	s.mux.HandleFunc("/stats", s.handleStats)
	s.mux.HandleFunc("/telegram", s.handleTelegram)
	s.mux.HandleFunc("/health", s.handleHealth)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "https://steered.dev")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) authenticate(r *http.Request) bool {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return false
	}
	return parts[1] == s.token
}

func (s *Server) handleTrack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	event.Country = r.Header.Get("CF-IPCountry")

	if event.Referrer != "" {
		if u, err := url.Parse(event.Referrer); err == nil {
			event.Referrer = u.Host
		}
	}

	s.store.Record(event)

	go s.telegram.Notify(event)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleInstall(w http.ResponseWriter, r *http.Request) {
	event := Event{
		Type:    "install_download",
		Country: r.Header.Get("CF-IPCountry"),
	}
	s.store.Record(event)
	go s.telegram.Notify(event)

	http.Redirect(w, r,
		"https://raw.githubusercontent.com/avvvet/steered/main/install.sh",
		http.StatusFound,
	)
}

func (s *Server) handleTelegram(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		Message struct {
			Text string `json:"text"`
			Chat struct {
				ID int64 `json:"id"`
			} `json:"chat"`
		} `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if payload.Message.Text == "/stats" {
		stats, err := s.store.GetStats()
		if err != nil {
			s.telegram.Send("error fetching stats")
			return
		}
		s.telegram.SendStats(stats)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if !s.authenticate(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	stats, err := s.store.GetStats()
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
