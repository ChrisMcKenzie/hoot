package chat

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
)

func TestUserMessageSend(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	rm := NewRoomManager(logger)
	room := rm.rooms[DefaultRoom]

	srv, cli := net.Pipe()

	user := &User{
		Name:    "test-user",
		conn:    srv,
		r:       bufio.NewReader(srv),
		w:       bufio.NewWriter(srv),
		rm:      rm,
		closeCh: make(chan struct{}),
	}
	go user.handle()
	defer user.Close()

	if err := user.JoinRoom(DefaultRoom); err != nil {
		t.Fatal(err)
	}

	msg := "hello\n"
	if _, err := cli.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}

	// read bytes but ignore them, to prevent close errors
	b := make([]byte, 10)
	cli.Read(b)

	msgs := room.GetMessages()
	if bytes.Equal([]byte(msgs[0].Content), []byte(msg)) {
		t.Fatalf("expected message to equal %s, got: %s", msg[0:len(msg)-1], msgs[1].Content)
	}
}

func TestUserRenameCommand(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	rm := NewRoomManager(logger)

	srv, cli := net.Pipe()

	user := &User{
		Name:    "test-user",
		conn:    srv,
		r:       bufio.NewReader(srv),
		w:       bufio.NewWriter(srv),
		rm:      rm,
		closeCh: make(chan struct{}),
	}
	go user.handle()

	if err := user.JoinRoom(DefaultRoom); err != nil {
		t.Fatal(err)
	}

	name := "chris"
	msg := fmt.Sprintf("/name %s\n", name)
	if _, err := cli.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}

	// read bytes but ignore them, to prevent close errors
	b := make([]byte, 64)
	cli.Read(b)

	if user.GetName() != name {
		t.Fatalf("expected user name to be changed to %s, got: %s", name, user.Name)
	}
}

func TestUserRoomChange(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)
	rm := NewRoomManager(logger)
	rm.CreateRoom("test-room")

	srv, cli := net.Pipe()

	user := &User{
		Name:    "test-user",
		conn:    srv,
		r:       bufio.NewReader(srv),
		w:       bufio.NewWriter(srv),
		rm:      rm,
		closeCh: make(chan struct{}),
	}
	go user.handle()

	if err := user.JoinRoom(DefaultRoom); err != nil {
		t.Fatal(err)
	}

	name := "test-room"
	msg := fmt.Sprintf("/room %s\n", name)
	if _, err := cli.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}

	// read bytes but ignore them, to prevent close errors
	b := make([]byte, 64)
	cli.Read(b)

	if user.CurrentRoom != name {
		t.Fatalf("expected user name to be changed to %s, got: %s", name, user.Name)
	}
}
