package vChat

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	dao *DAO
	hub *Hub
}

func NewService(dao *DAO, hub *Hub) *Service {
	return &Service{
		dao: dao,
		hub: hub,
	}
}

func (s *Service) CreateRoom(
	ctx context.Context,
	name string,
	maxParticipants int,
	createdBy string,
) (*CreateRoomResponse, error) {

	roomID := uuid.New().String()
	roomName := strings.TrimSpace(name)
	if roomName == "" {
		roomName = "Video Chat Room"
	}

	if maxParticipants <= 0 || maxParticipants > 10 {
		maxParticipants = 4 // Default max participants for WebRTC mesh
	}

	if createdBy == "" {
		createdBy = "guest"
	}

	room := &Room{
		ID:              roomID,
		Name:            roomName,
		CreatedBy:       createdBy,
		MaxParticipants: maxParticipants,
		IsActive:        true,
		CreatedAt:       time.Now(),
	}

	if err := s.dao.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	return &CreateRoomResponse{
		RoomID:          roomID,
		Name:            roomName,
		MaxParticipants: maxParticipants,
	}, nil
}

func (s *Service) GetRoom(ctx context.Context, roomID string) (*RoomResponse, error) {
	room, err := s.dao.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	participantCount := s.hub.GetParticipantCount(roomID)

	return &RoomResponse{
		RoomID:           room.ID,
		Name:             room.Name,
		CreatedBy:        room.CreatedBy,
		MaxParticipants:  room.MaxParticipants,
		ParticipantCount: participantCount,
		IsActive:         room.IsActive,
		CreatedAt:        room.CreatedAt,
	}, nil
}

func (s *Service) GetHub() *Hub {
	return s.hub
}
