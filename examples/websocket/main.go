package main

import (
	"fmt"
	"log"
	"time"

	"github.com/yourusername/goTap"
)

func main() {
	r := goTap.Default()

	// Create WebSocket hub for broadcasting
	hub := goTap.NewWebSocketHub()

	// Serve HTML page
	r.GET("/", func(c *goTap.Context) {
		c.Header("Content-Type", "text/html")
		c.String(200, htmlPage)
	})

	// WebSocket endpoint
	r.GET("/ws", func(c *goTap.Context) {
		c.WebSocket(func(ws *goTap.WebSocketConn) {
			// Register client
			hub.Register(ws)
			defer hub.Unregister(ws)

			log.Printf("Client connected. Total clients: %d", hub.ClientCount())

			// Send welcome message
			ws.SendJSON(goTap.H{
				"type":    "welcome",
				"message": "Connected to goTap WebSocket server",
				"time":    time.Now().Format(time.RFC3339),
			})

			// Read messages from client
			for {
				var msg map[string]interface{}
				if err := ws.ReadJSON(&msg); err != nil {
					log.Printf("Read error: %v", err)
					break
				}

				log.Printf("Received: %v", msg)

				// Echo message back
				ws.SendJSON(goTap.H{
					"type":    "echo",
					"message": msg,
					"time":    time.Now().Format(time.RFC3339),
				})
			}

			log.Printf("Client disconnected. Total clients: %d", hub.ClientCount())
		})
	})

	// Broadcast endpoint
	r.POST("/broadcast", func(c *goTap.Context) {
		message := c.PostForm("message")
		if message == "" {
			c.JSON(400, goTap.H{"error": "Message is required"})
			return
		}

		hub.BroadcastJSON(goTap.H{
			"type":    "broadcast",
			"message": message,
			"time":    time.Now().Format(time.RFC3339),
		})

		c.JSON(200, goTap.H{
			"status":  "sent",
			"clients": hub.ClientCount(),
		})
	})

	// Chat room example
	chatHub := goTap.NewWebSocketHub()

	r.GET("/chat", func(c *goTap.Context) {
		c.Header("Content-Type", "text/html")
		c.String(200, chatPage)
	})

	r.GET("/chat/ws", func(c *goTap.Context) {
		username := c.Query("username")
		if username == "" {
			username = "Anonymous"
		}

		c.WebSocket(func(ws *goTap.WebSocketConn) {
			chatHub.Register(ws)
			defer chatHub.Unregister(ws)

			// Announce user joined
			chatHub.BroadcastJSON(goTap.H{
				"type":     "system",
				"message":  fmt.Sprintf("%s joined the chat", username),
				"time":     time.Now().Format("15:04:05"),
				"username": "System",
			})

			log.Printf("%s joined. Total users: %d", username, chatHub.ClientCount())

			// Handle messages
			for {
				var msg map[string]interface{}
				if err := ws.ReadJSON(&msg); err != nil {
					break
				}

				// Broadcast to all users
				chatHub.BroadcastJSON(goTap.H{
					"type":     "message",
					"message":  msg["message"],
					"time":     time.Now().Format("15:04:05"),
					"username": username,
				})
			}

			// Announce user left
			chatHub.BroadcastJSON(goTap.H{
				"type":     "system",
				"message":  fmt.Sprintf("%s left the chat", username),
				"time":     time.Now().Format("15:04:05"),
				"username": "System",
			})

			log.Printf("%s left. Total users: %d", username, chatHub.ClientCount())
		})
	})

	// POS Terminal real-time updates
	posHub := goTap.NewWebSocketHub()

	r.GET("/pos/terminal", func(c *goTap.Context) {
		c.Header("Content-Type", "text/html")
		c.String(200, posTerminalPage)
	})

	r.GET("/pos/ws", func(c *goTap.Context) {
		terminalID := c.Query("terminal_id")
		if terminalID == "" {
			terminalID = "TERM-001"
		}

		c.WebSocket(func(ws *goTap.WebSocketConn) {
			posHub.Register(ws)
			defer posHub.Unregister(ws)

			log.Printf("POS Terminal %s connected", terminalID)

			ws.SendJSON(goTap.H{
				"type":        "connected",
				"terminal_id": terminalID,
				"status":      "online",
			})

			for {
				var msg map[string]interface{}
				if err := ws.ReadJSON(&msg); err != nil {
					break
				}

				log.Printf("POS %s: %v", terminalID, msg)

				// Process transaction and broadcast to all terminals
				if msg["type"] == "transaction" {
					posHub.BroadcastJSON(goTap.H{
						"type":           "transaction",
						"terminal_id":    terminalID,
						"amount":         msg["amount"],
						"product":        msg["product"],
						"transaction_id": fmt.Sprintf("TXN-%d", time.Now().Unix()),
						"time":           time.Now().Format(time.RFC3339),
					})
				}
			}

			log.Printf("POS Terminal %s disconnected", terminalID)
		})
	})

	// Status endpoint
	r.GET("/status", func(c *goTap.Context) {
		c.JSON(200, goTap.H{
			"websocket_clients": hub.ClientCount(),
			"chat_users":        chatHub.ClientCount(),
			"pos_terminals":     posHub.ClientCount(),
		})
	})

	log.Println("🚀 WebSocket Server starting on :5066")
	log.Println("Try these URLs:")
	log.Println("  http://localhost:5066/ - Basic WebSocket demo")
	log.Println("  http://localhost:5066/chat - Chat room")
	log.Println("  http://localhost:5066/pos/terminal - POS Terminal")
	log.Println("  http://localhost:5066/status - Server status")

	r.Run(":5066")
}

const htmlPage = `
<!DOCTYPE html>
<html>
<head>
    <title>goTap WebSocket Demo</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        #messages { border: 1px solid #ccc; height: 300px; overflow-y: scroll; padding: 10px; margin-bottom: 20px; background: #f9f9f9; }
        .message { margin: 5px 0; padding: 5px; border-left: 3px solid #667eea; }
        input, button { padding: 10px; margin: 5px; }
        button { background: #667eea; color: white; border: none; cursor: pointer; }
        button:hover { background: #5568d3; }
        .status { color: #666; font-size: 0.9em; }
    </style>
</head>
<body>
    <h1>🔌 goTap WebSocket Demo</h1>
    <div class="status" id="status">Connecting...</div>
    <div id="messages"></div>
    <input type="text" id="messageInput" placeholder="Type a message..." style="width: 70%;">
    <button onclick="sendMessage()">Send</button>
    <script>
        const ws = new WebSocket('ws://localhost:5066/ws');
        const messages = document.getElementById('messages');
        const status = document.getElementById('status');
        const input = document.getElementById('messageInput');

        ws.onopen = () => {
            status.textContent = '✅ Connected';
            status.style.color = 'green';
        };

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const div = document.createElement('div');
            div.className = 'message';
            div.innerHTML = '<strong>' + data.type + '</strong>: ' + JSON.stringify(data.message || data.message) + ' <span style="color:#999">' + data.time + '</span>';
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        };

        ws.onclose = () => {
            status.textContent = '❌ Disconnected';
            status.style.color = 'red';
        };

        ws.onerror = (error) => {
            status.textContent = '⚠️ Error';
            status.style.color = 'orange';
        };

        function sendMessage() {
            const message = input.value;
            if (message) {
                ws.send(JSON.stringify({ message: message }));
                input.value = '';
            }
        }

        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') sendMessage();
        });
    </script>
</body>
</html>
`

const chatPage = `
<!DOCTYPE html>
<html>
<head>
    <title>goTap Chat Room</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; background: #f0f0f0; }
        #chat { background: white; border-radius: 10px; padding: 20px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        #messages { height: 400px; overflow-y: scroll; border: 1px solid #ddd; padding: 15px; margin-bottom: 20px; background: #fafafa; }
        .message { margin: 10px 0; padding: 8px; border-radius: 5px; }
        .message.user { background: #e3f2fd; }
        .message.system { background: #fff3cd; text-align: center; font-style: italic; }
        .username { font-weight: bold; color: #667eea; }
        .time { color: #999; font-size: 0.8em; margin-left: 10px; }
        input { padding: 12px; width: 75%; border: 1px solid #ddd; border-radius: 5px; }
        button { padding: 12px 25px; background: #667eea; color: white; border: none; border-radius: 5px; cursor: pointer; margin-left: 10px; }
        button:hover { background: #5568d3; }
    </style>
</head>
<body>
    <div id="chat">
        <h1>💬 goTap Chat Room</h1>
        <div id="messages"></div>
        <input type="text" id="messageInput" placeholder="Type a message...">
        <button onclick="sendMessage()">Send</button>
    </div>
    <script>
        const username = prompt('Enter your username:') || 'Anonymous';
        const ws = new WebSocket('ws://localhost:5066/chat/ws?username=' + encodeURIComponent(username));
        const messages = document.getElementById('messages');
        const input = document.getElementById('messageInput');

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const div = document.createElement('div');
            div.className = 'message ' + (data.type === 'system' ? 'system' : 'user');
            if (data.type === 'system') {
                div.textContent = data.message;
            } else {
                div.innerHTML = '<span class="username">' + data.username + ':</span> ' + data.message + '<span class="time">' + data.time + '</span>';
            }
            messages.appendChild(div);
            messages.scrollTop = messages.scrollHeight;
        };

        function sendMessage() {
            const message = input.value.trim();
            if (message) {
                ws.send(JSON.stringify({ message: message }));
                input.value = '';
            }
        }

        input.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') sendMessage();
        });
    </script>
</body>
</html>
`

const posTerminalPage = `
<!DOCTYPE html>
<html>
<head>
    <title>POS Terminal</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 1000px; margin: 50px auto; padding: 20px; background: #1a1a1a; color: white; }
        .terminal { background: #2d2d2d; border-radius: 10px; padding: 30px; box-shadow: 0 4px 20px rgba(0,0,0,0.5); }
        h1 { color: #4CAF50; margin: 0 0 20px 0; }
        #transactions { height: 400px; overflow-y: scroll; border: 2px solid #4CAF50; padding: 15px; margin: 20px 0; background: #1a1a1a; font-family: 'Courier New', monospace; }
        .transaction { padding: 10px; margin: 5px 0; border-left: 4px solid #4CAF50; background: #2d2d2d; }
        .controls { display: grid; grid-template-columns: 1fr 1fr; gap: 15px; margin-top: 20px; }
        input, button { padding: 15px; border: none; border-radius: 5px; font-size: 16px; }
        input { background: #3d3d3d; color: white; }
        button { background: #4CAF50; color: white; cursor: pointer; font-weight: bold; }
        button:hover { background: #45a049; }
        .status { color: #4CAF50; font-weight: bold; padding: 10px; border: 2px solid #4CAF50; border-radius: 5px; text-align: center; margin-bottom: 20px; }
    </style>
</head>
<body>
    <div class="terminal">
        <h1>🏪 POS Terminal</h1>
        <div class="status" id="status">Offline</div>
        <div id="transactions"></div>
        <div class="controls">
            <input type="text" id="product" placeholder="Product name">
            <input type="number" id="amount" placeholder="Amount" step="0.01">
            <button onclick="processTransaction()" style="grid-column: 1 / -1;">Process Transaction</button>
        </div>
    </div>
    <script>
        const terminalID = 'TERM-' + Math.floor(Math.random() * 1000);
        const ws = new WebSocket('ws://localhost:5066/pos/ws?terminal_id=' + terminalID);
        const transactions = document.getElementById('transactions');
        const status = document.getElementById('status');

        ws.onopen = () => {
            status.textContent = '✅ Terminal ' + terminalID + ' - Online';
        };

        ws.onmessage = (event) => {
            const data = JSON.parse(event.data);
            const div = document.createElement('div');
            div.className = 'transaction';
            if (data.type === 'transaction') {
                div.innerHTML = '<strong>Transaction:</strong> ' + data.product + ' - $' + data.amount + '<br>' +
                               '<small>Terminal: ' + data.terminal_id + ' | ID: ' + data.transaction_id + ' | ' + new Date(data.time).toLocaleTimeString() + '</small>';
            } else {
                div.innerHTML = '<strong>' + data.type + ':</strong> ' + JSON.stringify(data);
            }
            transactions.insertBefore(div, transactions.firstChild);
        };

        ws.onclose = () => {
            status.textContent = '❌ Terminal ' + terminalID + ' - Offline';
        };

        function processTransaction() {
            const product = document.getElementById('product').value;
            const amount = document.getElementById('amount').value;
            if (product && amount) {
                ws.send(JSON.stringify({
                    type: 'transaction',
                    product: product,
                    amount: parseFloat(amount)
                }));
                document.getElementById('product').value = '';
                document.getElementById('amount').value = '';
            }
        }
    </script>
</body>
</html>
`
