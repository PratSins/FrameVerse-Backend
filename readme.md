This backend will have 2 features:

1. Video to cartoon (different styles) - gemini - record delay
2. Video chat normal (WebRTC mesh + Gorilla WebSocket signaling)

---

## 🚀 Roadmap & Future Features

### 1. Host Permission / Knock Admission (Waiting Room)
- **Host Knock Request**: When a user attempts to enter a video chat room using a Room ID, they will be placed in a waiting room state instead of joining immediately.
- **Creator Notification**: The room owner/creator receives a real-time admission prompt conveying the requester's name/identity.
- **Access Control**: The creator can explicitly **Admit** (granting WebRTC signaling access) or **Deny** (rejecting entry with feedback to the requester).