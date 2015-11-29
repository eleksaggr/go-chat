package main

import (
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/zillolo/chat/app"
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

var channels []*app.Channel

var id uint32 = 0

var wg sync.WaitGroup

func handler(w http.ResponseWriter, r *http.Request) {
	log.Info("Entered handler.")
	// Get the parameters from the url
	vars := mux.Vars(r)

	// Try to connect to the clients websocket.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Warning("Couldn't establish connection to WebSocket.")
		return
	}

	user := app.NewUser(id, conn)
	id = id + 1

	// Read the username from the client.
	username, err := user.Read()
	if err != nil {
		log.Error("Couldn't read username from client.")
		return
	}
	user.Nickname = username

	// Add the user to his selected channel.
	addToChannel(user, vars["channel"])

	channel, err := getChannel(vars["channel"])
	if err != nil {
		log.Error("A non-existent channel was called.")
		return
	}

	wg.Add(1)
	go handleUser(user)

	wg.Wait()
	channel.Remove(user.Id)
	log.Info("User was removed from channel.")

	user.Close()
}

func handleUser(user *app.User) {
	defer wg.Done()
	if user == nil {
		return
	}

	channel, err := getChannelForUser(user)
	if err != nil {
		log.Error("User not in channel anymore.")
		return
	}
	for {
		msg, err := user.Read()
		if err != nil {
			log.Error(fmt.Sprintf("An error happend during read for user: %s", user.Nickname))
			return
		}

		log.Info(fmt.Sprintf("%s: %s", user.Nickname, msg))
		channel.Message <- msg
	}
}

func broadcast() {
	for {
		for _, channel := range channels {
			select {
			case msg, ok := <-channel.Message:
				if ok {
					log.Info(fmt.Sprintf("Writing a message to channel %s", channel.Name))
					for _, user := range channel.Members {
						err := user.Write(msg)
						if err != nil {
							log.Error(fmt.Sprintf("An error happend during write for user: %s", user.Nickname))
						}
					}
				}
			}
		}
	}
}

// listUsers prints all users in the channel to stdout.
func listUsers(channel *app.Channel) {
	fmt.Println("Current users in channel:")
	for _, user := range channel.Members {
		fmt.Printf("%d. %s\n", user.Id, user.Nickname)
	}
}

// getChannel gets a channel by name from the channel list.
// If it does not exist it returns an error, else the channel.
func getChannel(channelName string) (*app.Channel, error) {
	for _, channel := range channels {
		if channel.Name == channelName {
			return channel, nil
		}
	}
	return nil, errors.New("The channel was not found.")
}

func getChannelForUser(user *app.User) (*app.Channel, error) {
	for _, channel := range channels {
		for _, u := range channel.Members {
			if user == u {
				return channel, nil
			}
		}
	}
	return nil, errors.New("No channel found for the user.")
}

// addToChannel adds user to the channel with the name channelName, if it exists.
// If it does not, the channel will be created and the user added.
// If the user is nil, the function will return an error.
func addToChannel(user *app.User, channelName string) error {
	if user == nil {
		log.Error("Tried to add a nil-user to a channel.")
		return errors.New("User was nil.")
	}

	// Check if the channel already exists, and if so add the user.
	channel, err := getChannel(channelName)
	if err != nil {
		// If we couldn't find the channel, create it and add the user.
		channel := app.NewChannel(channelName)
		channel.Add(user)

		log.Info(fmt.Sprintf("Adding new channel: %s", channelName))
		log.Info(fmt.Sprintf("Adding user %s to channel %s", user.Nickname, channel.Name))
		channels = append(channels, channel)
	} else {
		log.Info(fmt.Sprintf("Adding user %s to channel %s", user.Nickname, channel.Name))
		channel.Add(user)
	}
	return nil
}

func main() {
	log.Info("Server started on port 8080.")

	go broadcast()
	log.Info("Broadcast started.")

	router := mux.NewRouter()
	log.Info("t")
	router.HandleFunc("/{channel}", handler)
	log.Info("suchhia")
	http.ListenAndServe(":8080", router)
}
