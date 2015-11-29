package app

import (
  "errors"
  "github.com/gorilla/websocket"
)

type User struct {
  Id  uint32
  Nickname  string
  conn  *websocket.Conn
}

// func (user *User) Write(message string) error {
//
// }

func NewUser(id uint32, conn *websocket.Conn) (*User) {
  return &User{Id: id, conn: conn}
}

func (user *User) Write(message string) (error) {
  if user.conn == nil {
		return errors.New("Tried writing to a nil-socket.")
	}

  user.conn.WriteMessage(websocket.TextMessage, []byte(message))
  return nil
}

func (user *User) Read() (string, error) {
  if user.conn == nil {
		return "", errors.New("Tried reading from a nil-socket.")
	}

	messageType, p, err := user.conn.ReadMessage()
	if err != nil {
		return "", err
	}

	if messageType != websocket.TextMessage {
		return "", errors.New("Received an invalid message type from the client.")
	}

	return string(p), nil
}

func (user *User) Close() {
  user.conn.Close()
}
