# Real-time Chat Application

A modern, feature-rich real-time chat application built with **Go WebSockets** and **HTML5**. Send messages, share files, and see typing indicators in real-time across multiple connected clients.

## Features

- **📱 Real-time Messaging** - Instant message delivery to all connected users
- **⌨️ Typing Indicators** - See when other users are typing
- **👥 User Management** - Set custom usernames and unique user IDs
- **📊 Live User Count** - See how many users are connected
- **🔄 Auto-Reconnect** - Automatic reconnection on connection loss
- **💻 Cross-Browser Support** - Works on all modern browsers

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Web Browser (Client)                     │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  client.html - Chat Interface                         │  │
│  │  ├─ Message Display Area                              │  │
│  │  ├─ Typing Indicator Animation                        │  │
│  │  ├─ File Upload/Download                             │  │
│  │  └─ Username Configuration                           │  │
│  └───────────────────────────────────────────────────────┘  │
│                         ↕ WebSocket                          │
└─────────────────────────────────────────────────────────────┘
                              ↕
┌─────────────────────────────────────────────────────────────┐
│                   Go Server (Backend)                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  main.go - WebSocket Server                          │  │
│  │  ├─ Hub (Message Broker)                              │  │
│  │  ├─ Client Manager                                   │  │
│  │  ├─ ReadPump (Receive Messages)                      │  │
│  │  ├─ WritePump (Send Messages)                        │  │
│  │  └─ Gorilla WebSocket Handler                        │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Project Structure

```
f:\projects\chat/
├── main.go                 # Go WebSocket server
├── client.html             # Web-based chat interface
├── go.mod                  # Go module dependencies
├── go.sum                  # Go module checksums
├── .gitignore              # Git ignore file
└── README.md               # This file
```

## Quick Start

### Prerequisites
- **Go 1.16+** installed on your system
- Modern web browser (Chrome, Firefox, Safari, Edge)

### Installation

1. **Clone or navigate to the project directory:**
   ```bash
   cd f:\projects\chat
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Start the server:**
   ```bash
   go run main.go
   ```

   You should see:
   ```
   ========================================
   Chat server starting on port :8080
   WebSocket endpoint: ws://localhost:8080/ws
   Chat client: http://localhost:8080/
   ========================================
   Server is ready! Open browser to test.
   ========================================
   ```

4. **Open in browser:**
   - Open **two or more browser windows** (or use incognito mode)
   - Visit: `http://localhost:8080/`
   - Set different usernames in each window
   - Start chatting!

## 💡 How to Use

### Sending Messages
1. Type your message in the text input field
2. Press **Enter** or click the **Send** button
3. Your message appears in all connected clients' chat windows

### Typing Indicator
- As you type, other users will see **"User is typing..."** with an animated indicator
- The indicator disappears after 5 seconds of inactivity


### Managing Users
- Enter your username in the text field at the top
- Click **Set Username** or press Enter
- Your unique User ID is generated automatically
- The header shows how many users are connected

## 🔧 Technical Details

### Technologies Used

| Component | Technology | Version |
|-----------|-----------|---------|
| **Backend** | Go (Golang) | 1.16+ |
| **WebSocket** | Gorilla WebSocket | Latest |
| **Frontend** | HTML5/CSS3/JavaScript | ES6+ |
| **Protocol** | WebSocket (RFC 6455) | Latest |

### Message Types

The application supports three types of messages:

#### 1. **Text Messages**
```json
{
  "type": "message",
  "userID": "user_abc123",
  "username": "John",
  "content": "Hello everyone!",
  "timestamp": 1762886360
}
```

#### 2. **Typing Indicators**
```json
{
  "type": "typing",
  "userID": "user_abc123",
  "username": "John",
  "timestamp": 1762886360
}
```

#### 3. **Client Count Updates**
```json
{
  "type": "client_count",
  "clientCount": 3,
  "timestamp": 1762886360
}
```

## Example Scenarios

```
User 1 (Browser 1)          User 2 (Browser 2)
   ↓                              ↓
[Connects] ─────────────→ [Connected - 2 users]
   ↓                              ↓
[Types "Hi!"] ─────────→ [Typing indicator appears]
   ↓                              ↓
[Sends message] ──────→ [Message appears]
   ↓                              ↓
[Types reply] ────────→ [Typing indicator appears]
   ↓                              ↓
[Sends message] ──────→ [Message appears]
```

## Learning Resources

This project demonstrates:
- ✅ WebSocket bidirectional communication
- ✅ Goroutine concurrency patterns
- ✅ Channel-based synchronization
- ✅ Real-time message broadcasting
- ✅ Base64 file encoding/decoding
- ✅ JSON serialization
- ✅ Frontend-backend integration

## Future Enhancements

- [ ] User authentication & login system
- [ ] Message persistence (database)
- [ ] Private messaging between users
- [ ] Message search functionality
- [ ] User blocking/muting
- [ ] Emoji reactions
- [ ] Voice/video chat integration
- [ ] Message editing/deletion
- [ ] Group creation & management
- [ ] Admin dashboard
