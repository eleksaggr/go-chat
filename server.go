package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/zillolo/chat/app"
	"github.com/zillolo/loggo"
	"net/http"
	"os"
)

var log = loggo.NewLog(os.Stderr)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

var channels []*app.Channel

var id uint32 = 0

func handler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error("Couldn't establish connection to WebSocket.")
		return
	}

	// Check if Channel exists
	var channel *app.Channel

	exists := false
	for _, c := range channels {
		if c.Name == vars["channel"] {
			log.Info("The channel exists already.")
			channel = c
			exists = true
		}
	}

	if !exists {
		channel = app.NewChannel(vars["channel"])

		log.Info(fmt.Sprintf("Adding new channel %s", vars["channel"]))
		channels = append(channels, channel)
	}

	channel.Add(app.User{id, "TestUser"})
	id = id + 1

	listUsers(channel)

	msg := fmt.Sprintf("You have joined channel %s", vars["channel"])

	conn.WriteMessage(websocket.TextMessage, []byte(msg))

	channel.Remove(id)
	listUsers(channel)

	log.Info("WebSocket finished. Closing.")
	conn.Close()
}

func listUsers(channel *app.Channel) {
	fmt.Println("Current users in channel:")
	for _, user := range channel.Members {
		fmt.Printf("%d. %s\n", user.Id, user.Nickname)
	}
}

func main() {
	log.Info("Server started on port 8080.")

	router := mux.NewRouter()
	router.HandleFunc("/{channel}", handler)
	http.ListenAndServe(":8080", router)
}
