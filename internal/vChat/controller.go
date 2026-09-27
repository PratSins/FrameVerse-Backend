package vChat

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/PratSins/FrameVerse-Backend/pkg/authclient"
	"github.com/PratSins/FrameVerse-Backend/pkg/middleware"
	"github.com/go-chi/chi/v5"
)

type Controller struct {
	service    *Service
	authClient *authclient.Client
}

func NewController(service *Service, authClient *authclient.Client) *Controller {
	return &Controller{
		service:    service,
		authClient: authClient,
	}
}

func (c *Controller) MountRoutes(r chi.Router, authGuard func(http.Handler) http.Handler) {
	// REST API group - Strictly protected
	r.Route("/api/v1/vchat", func(r chi.Router) {
		r.Use(authGuard)
		r.Post("/rooms", c.CreateRoom)
		r.Get("/rooms/{roomID}", c.GetRoom)
	})

	// WebSocket Signaling endpoint - Strictly token-authenticated
	r.Get("/ws/vchat/rooms/{roomID}", c.HandleWebSocket)
}

func (c *Controller) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	userID := middleware.GetUserID(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized: login required to create rooms")
		return
	}

	resp, err := c.service.CreateRoom(r.Context(), req.Name, req.MaxParticipants, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create room: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (c *Controller) GetRoom(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		writeError(w, http.StatusBadRequest, "missing room id")
		return
	}

	room, err := c.service.GetRoom(r.Context(), roomID)
	if err != nil {
		if errors.Is(err, ErrRoomNotFound) {
			writeError(w, http.StatusNotFound, "room not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to get room: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, room)
}

func (c *Controller) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "roomID")
	if roomID == "" {
		http.Error(w, "missing room id", http.StatusBadRequest)
		return
	}

	// Strictly authenticate via token
	claims, err := middleware.WebSocketAuth(c.authClient, r)
	if err != nil || claims == nil {
		log.Printf("[vChat] WebSocket unauthorized connection attempt for room %s", roomID)
		http.Error(w, `{"error":"unauthorized: valid login token required to join vChat rooms"}`, http.StatusUnauthorized)
		return
	}

	// Validate room exists
	room, err := c.service.GetRoom(r.Context(), roomID)
	if err != nil || !room.IsActive {
		http.Error(w, "room not found or inactive", http.StatusNotFound)
		return
	}

	// Check if room is at max capacity
	if room.ParticipantCount >= room.MaxParticipants {
		http.Error(w, "room is full", http.StatusForbidden)
		return
	}

	clientID := claims.UserID
	clientName := claims.Email
	if claims.Email == "" {
		clientName = claims.UserID
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[vChat] WebSocket upgrade failed for room %s: %v", roomID, err)
		return
	}

	client := &Client{
		ID:     clientID,
		Name:   clientName,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		Hub:    c.service.GetHub(),
	}

	client.Hub.register <- client

	go client.WritePump()
	go client.ReadPump()
}

func writeJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	writeJSON(w, statusCode, map[string]string{"error": message})
}
