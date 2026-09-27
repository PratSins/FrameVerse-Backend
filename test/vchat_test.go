package test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/PratSins/FrameVerse-Backend/internal/vChat"
)

func TestVChatHub_RegisterAndRouting(t *testing.T) {
	hub := vChat.NewHub()
	go hub.Run()

	roomID := "room-test-123"

	clientA := &vChat.Client{
		ID:     "user-a",
		Name:   "Alice",
		RoomID: roomID,
		Send:   make(chan []byte, 10),
		Hub:    hub,
	}

	clientB := &vChat.Client{
		ID:     "user-b",
		Name:   "Bob",
		RoomID: roomID,
		Send:   make(chan []byte, 10),
		Hub:    hub,
	}

	// Register Alice
	hub.RegisterClientForTest(clientA)
	time.Sleep(10 * time.Millisecond)

	if hub.GetParticipantCount(roomID) != 1 {
		t.Errorf("expected 1 participant in room, got %d", hub.GetParticipantCount(roomID))
	}

	// Register Bob (Hub sends initial room-info greeting)
	hub.RegisterClientForTest(clientB)
	time.Sleep(10 * time.Millisecond)

	if hub.GetParticipantCount(roomID) != 2 {
		t.Errorf("expected 2 participants in room, got %d", hub.GetParticipantCount(roomID))
	}

	// Drain Bob's initial room-info greeting message
	select {
	case greeting := <-clientB.Send:
		var roomInfo vChat.SignalingMessage
		if err := json.Unmarshal(greeting, &roomInfo); err != nil {
			t.Fatalf("failed to unmarshal greeting: %v", err)
		}
		if roomInfo.Type != vChat.MsgTypeRoomInfo {
			t.Errorf("expected initial message MsgTypeRoomInfo, got %s", roomInfo.Type)
		}
	default:
		t.Fatal("expected clientB to receive initial room-info message")
	}

	// Send direct SDP Offer from Alice to Bob
	offerMsg := &vChat.SignalingMessage{
		Type:     vChat.MsgTypeOffer,
		RoomID:   roomID,
		SenderID: "user-a",
		TargetID: "user-b",
		Payload: map[string]string{
			"sdp": "v=0\r\no=- 12345 2 IN IP4 127.0.0.1...",
		},
	}

	hub.BroadcastMessageForTest(offerMsg)
	time.Sleep(10 * time.Millisecond)

	select {
	case msgBytes := <-clientB.Send:
		var received vChat.SignalingMessage
		if err := json.Unmarshal(msgBytes, &received); err != nil {
			t.Fatalf("failed to unmarshal received message: %v", err)
		}
		if received.Type != vChat.MsgTypeOffer {
			t.Errorf("expected MsgTypeOffer, got %s", received.Type)
		}
		if received.SenderID != "user-a" {
			t.Errorf("expected SenderID user-a, got %s", received.SenderID)
		}
	default:
		t.Fatal("expected clientB to receive the routed offer message")
	}

	// Unregister Bob
	hub.UnregisterClientForTest(clientB)
	time.Sleep(10 * time.Millisecond)

	if hub.GetParticipantCount(roomID) != 1 {
		t.Errorf("expected 1 participant after Bob left, got %d", hub.GetParticipantCount(roomID))
	}
}
