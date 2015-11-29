package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/zillolo/loggo"
	"net/http"
	"os"
	"sync"
)

var log = loggo.NewLog(os.Stderr)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var rooms []*Room
var roomsMutex sync.Mutex

type Room struct {
	Name        string
	clients     map[string]Client
	clientMutex sync.Mutex
	queue       chan string
}

// NewRoom creates a new room with the name name and initializes it.
func NewRoom(name string) *Room {
	room := new(Room)
	room.Name = name
	room.clients = make(map[string]Client)
	room.queue = make(chan string)

	go room.Broadcast()

	return room
}

// Join makes the client join the room.
// Should the client-pointer be nil, an error will be returned.
func (room *Room) Join(client *Client) error {
	if client == nil {
		return errors.New("nil-client cannot join a room.")
	}

	// Lock the client map for the room.
	room.clientMutex.Lock()

	if _, exists := room.clients[client.Nickname]; !exists {
		room.clients[client.Nickname] = *client
		if err := client.SetRoom(room); err != nil {
			return err
		}
	}

	room.clientMutex.Unlock()
	return nil
}

// Leave makes the client leave the room.
// Should the client-pointer be nil, an error will be returned.
func (room *Room) Leave(client *Client) error {
	if client != nil {
		return errors.New("Nil-client cannot leave the room.")
	}

	room.clientMutex.Lock()
	delete(room.clients, client.Nickname)
	room.clientMutex.Unlock()
	return nil
}

// AddMessage adds a message to the message queue of the room.
func (room *Room) AddMessage(message string) {
	room.queue <- message
}

func (room *Room) Broadcast() {
	for {
		message := <-room.queue
		for _, client := range room.clients {
			if err := client.Write(message); err != nil {
				if err = client.Exit(); err != nil {
					log.Error("Client exited during write.")
					break
				}
			}
		}
	}
}

type Client struct {
	Nickname string
	conn     *websocket.Conn
	room     *Room
}

// NewClient creates a new client with the nickname name and the socket conn.
// Should the connection-pointer be nil an error will be returned.
func NewClient(name string, conn *websocket.Conn) (*Client, error) {
	if conn == nil {
		return nil, errors.New("Cannot create a client with a nil-socket.")
	}

	return &Client{Nickname: name, conn: conn}, nil
}

// Read reads a message from the client.
// Should there be an error during the read, it will be returned.
func (client *Client) Read() (string, error) {
	messageType, message, err := client.conn.ReadMessage()
	if err != nil {
		return "", err
	}

	if messageType != websocket.TextMessage {
		return "", errors.New("Received a non-text message.")
	}

	return string(message), nil
}

// Write writes a message to the client.
// Should there be an error during write, it will be returned.
func (client *Client) Write(message string) error {
	err := client.conn.WriteMessage(websocket.TextMessage, []byte(message))
	if err != nil {
		return err
	}
	return nil
}

// Exit removes a client from his room and closes the websocket.
func (client *Client) Exit() error {
	err := client.room.Leave(client)
	if err != nil {
		return err
	}
	client.conn.Close()
	return nil
}

// SetRoom sets the room a client belongs to. It will NOT make the client join
// a room.
// Should the room be a nil-pointer, an error will be returned.
func (client *Client) SetRoom(room *Room) error {
	if room == nil {
		return errors.New("Client cannot join nil-room.")
	}

	client.room = room
	return nil
}

// Room is the room the client is currently in.
func (client *Client) Room() *Room {
	return client.room
}

func handler(w http.ResponseWriter, r *http.Request) {
	log.Info("New client has connected.")
	vars := mux.Vars(r)

	// Wait until we get a websocket connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warning("Couldn't connect to WebSocket.")
		return
	}

	go func() {
		// Read the nickname from the client.
		_, name, err := conn.ReadMessage()
		if err != nil {
			log.Error("Couldn't read nickname from client.")
			return
		}

		// Create a new client.
		client, err := NewClient(string(name), conn)
		if err != nil {
			log.Error("Couldn't create a new client.")
			return
		}

		// Check if the room the client is joining already exists
		// if so, join it, else create it and then join it.
		exists := false
		for _, room := range rooms {
			if room.Name == vars["room"] {
				room.Join(client)
				room.AddMessage(fmt.Sprintf("User %s has joined the channel."))
				log.Info(fmt.Sprintf("User %s joined channel %s", client.Nickname, room.Name))
				exists = true
			}
		}

		if !exists {
			room := NewRoom(vars["room"])
			log.Info(fmt.Sprintf("Created new channel: %s", vars["room"]))

			room.Join(client)
			room.AddMessage(fmt.Sprintf("User %s has joined the channel.", client.Nickname))
			log.Info(fmt.Sprintf("User %s joined channel %s", client.Nickname, room.Name))

			roomsMutex.Lock()
			rooms = append(rooms, room)
			roomsMutex.Unlock()
		}

		for {
			message, err := client.Read()
			if err != nil {
				err = client.Exit()
				if err != nil {
					log.Error("Error during exit from client.")
				}
				log.Error("Error during read from client.")
				break
			}
			client.Room().AddMessage(fmt.Sprintf("%s: %s", client.Nickname, message))
			log.Info(fmt.Sprintf("%s: %s", client.Nickname, message))
		}
	}()
}

func main() {
	log.Info("Server started on port :8080")

	router := mux.NewRouter()
	router.HandleFunc("/{room}", handler)
	http.ListenAndServe(":8080", router)
}
