package vChat

import (
	"time"
)

type Room struct {
	ID              string     `json:"id" bson:"_id"`
	Name            string     `json:"name" bson:"name"`
	CreatedBy       string     `json:"created_by" bson:"created_by"`
	MaxParticipants int        `json:"max_participants" bson:"max_participants"`
	IsActive        bool       `json:"is_active" bson:"is_active"`
	CreatedAt       time.Time  `json:"created_at" bson:"created_at"`
	EndedAt         *time.Time `json:"ended_at,omitempty" bson:"ended_at,omitempty"`
}

type SignalingMessageType string

const (
	MsgTypeJoin         SignalingMessageType = "join"
	MsgTypeJoined       SignalingMessageType = "peer-joined"
	MsgTypeOffer        SignalingMessageType = "offer"
	MsgTypeAnswer       SignalingMessageType = "answer"
	MsgTypeICECandidate SignalingMessageType = "ice-candidate"
	MsgTypeLeave        SignalingMessageType = "leave"
	MsgTypePeerLeft     SignalingMessageType = "peer-left"
	MsgTypeRoomInfo     SignalingMessageType = "room-info"
	MsgTypeError        SignalingMessageType = "error"
)

type SignalingMessage struct {
	Type       SignalingMessageType `json:"type"`
	RoomID     string               `json:"room_id,omitempty"`
	SenderID   string               `json:"sender_id,omitempty"`
	SenderName string               `json:"sender_name,omitempty"`
	TargetID   string               `json:"target_id,omitempty"`
	Payload    interface{}          `json:"payload,omitempty"`
}

type CreateRoomRequest struct {
	Name            string `json:"name"`
	MaxParticipants int    `json:"max_participants"`
}

type CreateRoomResponse struct {
	RoomID          string `json:"room_id"`
	Name            string `json:"name"`
	MaxParticipants int    `json:"max_participants"`
}

type RoomResponse struct {
	RoomID           string    `json:"room_id"`
	Name             string    `json:"name"`
	CreatedBy        string    `json:"created_by"`
	MaxParticipants  int       `json:"max_participants"`
	ParticipantCount int       `json:"participant_count"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
}
