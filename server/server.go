package server

import (
	"log"
	"net"

	"github.com/chrismckenzie/hoot/chat"
)

// HootServer handles accepting of tcp connections and user creation
type HootServer struct {
	Addr        string
	RoomManager *chat.RoomManager
}

func NewHootServer(addr string, rm *chat.RoomManager) *HootServer {
	return &HootServer{
		Addr:        addr,
		RoomManager: rm,
	}
}

// ListenAndServe attempts to listen on the address set in the server and
// starts accepting new connections, creating a new user per connection.
func (hs *HootServer) ListenAndServe() error {
	l, err := net.Listen("tcp", hs.Addr)
	if err != nil {
		return err
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}

		go func() {
			log.Printf("new connection from %s", conn.RemoteAddr())
			user := chat.NewUser(conn, hs.RoomManager)
			log.Printf("%s joined", user.Name)

			if err := user.JoinRoom(chat.DefaultRoom); err != nil {
				log.Printf("unable to join room: %s", err)
			}
		}()
	}

	return nil
}
