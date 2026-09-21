package vChat

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/PratSins/FrameVerse-Backend/pkg/authclient"
	"github.com/PratSins/FrameVerse-Backend/pkg/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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

func (c *Controller) MountRoutes(r chi.Router, optionalAuth func(http.Handler) http.Handler) {
	// REST API group
	r.Route("/api/v1/vchat", func(r chi.Router) {
		r.Use(optionalAuth)
		r.Post("/rooms", c.CreateRoom)
		r.Get("/rooms/{roomID}", c.GetRoom)
	})

	// WebSocket Signaling endpoint
	r.Get("/ws/vchat/rooms/{roomID}", c.HandleWebSocket)
}

func (c *Controller) CreateRoom(w http.ResponseWriter, r *http.Request) {
	var req CreateRoomRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	userID := middleware.GetUserID(r.Context())

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

	// Authenticate via token if present
	var clientID string
	var clientName string

	claims, err := middleware.WebSocketAuth(c.authClient, r)
	if err == nil && claims != nil {
		clientID = claims.UserID
		clientName = claims.Email
	} else {
		// Anonymous Guest
		guestUUID := uuid.New().String()[:8]
		clientID = fmt.Sprintf("guest-%s", guestUUID)
		clientName = fmt.Sprintf("Guest_%s", guestUUID)
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
