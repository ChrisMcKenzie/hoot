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

func TestRoomJoinAndLeave(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	user := &User{
		Name: "test-user",
		conn: cli,
		r:    bufio.NewReader(cli),
		w:    bufio.NewWriter(cli),
	}

	room := NewRoom("test", logger)
	room.AddUser(user)

	if _, ok := room.GetUsers()[user.Name]; !ok {
		t.Fatalf("expected users to contain our \"test-user\" user")
	}

	if room.GetMessages()[0].Content == fmt.Sprintf("%s joined", user.Name) {
		t.Fatalf("expected first message in room to be user join message, got: %s", room.msgs[0].Content)
	}

	room.RemoveUser(user)
	users := room.GetUsers()
	if _, ok := users[user.Name]; ok {
		t.Fatalf("expected user to be removed from room.")
	}

	if room.GetMessages()[1].Content == fmt.Sprintf("%s left", user.Name) {
		t.Fatalf("expected second message in room to be user leave message, got: %s", room.msgs[1].Content)
	}
}

func TestBroadCast(t *testing.T) {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	srv, cli := net.Pipe()
	defer srv.Close()
	defer cli.Close()

	user := &User{
		Name: "test-user",
		conn: srv,
		r:    bufio.NewReader(srv),
		w:    bufio.NewWriter(srv),
	}

	room := NewRoom("test", logger)
	room.AddUser(user)

	msg := NewMessage("hello, world", serverUser)
	go func() {
		room.BroadCast(msg)
	}()

	result := make(chan error)
	go func(result chan error) {
		r := bufio.NewReader(cli)
		data, _, err := r.ReadLine()
		if bytes.Equal(data, []byte(msg.String())) {
			result <- fmt.Errorf("expected msg to equal: %s, got: %s", msg.String(), string(data))
		}
		result <- err
	}(result)

	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	}
}
