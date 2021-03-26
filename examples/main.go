package main

import (
	"fmt"
	"log"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/sujit-baniya/ws"
)

// Basic chat message object
type MessageObject struct {
	Data  string `json:"data"`
	From  string `json:"from"`
	Event string `json:"event"`
	To    string `json:"to"`
}

func main() {

	// The key for the map is message.to
	clients := make(map[string]string)

	// Start a new Fiber application
	app := fiber.New()

	// Setup the middleware to retrieve the data sent in first GET request
	app.Use("/ws", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Multiple event handling supported
	ws.On(ws.EventConnect, func(ep *ws.EventPayload) {
		fmt.Println(fmt.Sprintf("Connection event 1 - User: %s", ep.SocketAttributes["user_id"]))
	})

	// Custom event handling supported
	ws.On("CUSTOM_EVENT", func(ep *ws.EventPayload) {
		fmt.Println(fmt.Sprintf("Custom event - User: %s", ep.SocketAttributes["user_id"]))
		// --->

		// DO YOUR BUSINESS HERE

		// --->
	})

	// On message event
	ws.On(ws.EventMessage, func(ep *ws.EventPayload) {

		fmt.Println(fmt.Sprintf("Message event - User: %s - Message: %s", ep.SocketAttributes["user_id"], string(ep.Data)))

		message := MessageObject{}

		// Unmarshal the json message
		// {
		//  "from": "<user-id>",
		//  "to": "<recipient-user-id>",
		//  "event": "CUSTOM_EVENT",
		//  "data": "hello"
		//}
		err := json.Unmarshal(ep.Data, &message)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Fire custom event based on some
		// business logic
		if message.Event != "" {
			ep.Kws.Fire(message.Event, []byte(message.Data))
		}

		// Emit the message directly to specified user
		err = ep.Kws.EmitTo(clients[message.To], ep.Data)
		if err != nil {
			fmt.Println(err)
		}
	})

	// On disconnect event
	ws.On(ws.EventDisconnect, func(ep *ws.EventPayload) {
		// Remove the user from the local clients
		delete(clients, ep.SocketAttributes["user_id"])
		fmt.Println(fmt.Sprintf("Disconnection event - User: %s", ep.SocketAttributes["user_id"]))
	})

	// On close event
	// This event is called when the server disconnects the user actively with .Close() method
	ws.On(ws.EventClose, func(ep *ws.EventPayload) {
		// Remove the user from the local clients
		delete(clients, ep.SocketAttributes["user_id"])
		fmt.Println(fmt.Sprintf("Close event - User: %s", ep.SocketAttributes["user_id"]))
	})

	// On error event
	ws.On(ws.EventError, func(ep *ws.EventPayload) {
		fmt.Println(fmt.Sprintf("Error event - User: %s", ep.SocketAttributes["user_id"]))
	})

	app.Get("/ws/:id", ws.New(func(kws *ws.Websocket) {

		// Retrieve the user id from endpoint
		userId := kws.Params("id")

		// Add the connection to the list of the connected clients
		// The UUID is generated randomly and is the key that allow
		// ws to manage Emit/EmitTo/Broadcast
		clients[userId] = kws.UUID

		// Every websocket connection has an optional session key => value storage
		kws.SetAttribute("user_id", userId)

		//Broadcast to all the connected users the newcomer
		kws.Broadcast([]byte(fmt.Sprintf("New user connected: %s and UUID: %s", userId, kws.UUID)), true)
		//Write welcome message
		kws.Emit([]byte(fmt.Sprintf("Hello user: %s with UUID: %s", userId, kws.UUID)))
	}))
	app.Static("/websocket", "websocket.html")
	app.Static("/websocket2", "websocket2.html")
	log.Fatal(app.Listen(":3000"))
}
