package chat

import (
	"fmt"
	"log"
	"sync"
)

const DefaultRoom = "lobby"

// RoomManager keeps track of open rooms, and allows for safe creation and deletion
// of rooms
type RoomManager struct {
	rooms map[string]*Room
	mu    sync.Mutex

	logger *log.Logger
}

func NewRoomManager(logger *log.Logger) *RoomManager {
	rm := &RoomManager{
		rooms:  make(map[string]*Room),
		logger: logger,
	}
	rm.CreateRoom(DefaultRoom)
	return rm
}

// CreateRoom will create a new room and add it to the RoomManager if room already
// exists it will no-op
func (rm *RoomManager) CreateRoom(name string) {
	if _, ok := rm.rooms[name]; ok {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.rooms[name] = NewRoom(name, rm.logger)
}

// DeleteRoom will remove the room from the manager and will no-op if it doesn't
// exist.
func (rm *RoomManager) DeleteRoom(name string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	delete(rm.rooms, name)
}

// JoinRoom will add the given user to the given room and error if room does
// not exist
func (rm *RoomManager) JoinRoom(u *User, name string) error {
	r, ok := rm.rooms[name]
	if !ok {
		return fmt.Errorf("room \"%s\" does not exist", name)
	}

	rm.logger.Printf("user \"%s\" joined room \"%s\"", u.GetName(), name)
	r.AddUser(u)
	return nil
}

// LeaveRoom will remove the given user from a room. will no-op if room does
// not exist or if user is not in room.
func (rm *RoomManager) LeaveRoom(u *User, name string) {
	r, ok := rm.rooms[name]
	if !ok {
		return
	}

	r.RemoveUser(u)
}

// Broadcast will send a message to all users in the given room will fail if
// room does not exist
func (rm *RoomManager) Broadcast(name string, msg Message) error {
	r, ok := rm.rooms[name]
	if !ok {
		return fmt.Errorf("room \"%s\" does not exist", name)
	}

	r.BroadCast(msg)
	return nil
}

func (rm *RoomManager) GetRooms() map[string]*Room {
	return rm.rooms
}

func (rm *RoomManager) RoomExists(name string) bool {
	_, ok := rm.rooms[name]
	return ok
}

type Room struct {
	Name string

	msgs   []Message
	msgsMu sync.RWMutex
	users  sync.Map

	logger *log.Logger
}

func NewRoom(name string, logger *log.Logger) *Room {
	r := &Room{
		Name:   name,
		msgs:   make([]Message, 0),
		logger: logger,
	}

	return r
}

// AddUser will safely add user to room.
func (r *Room) AddUser(u *User) {
	_, ok := r.users.Load(u.GetName())

	if !ok {
		r.users.Store(u.GetName(), u)
		msg := NewMessage(fmt.Sprintf("%s joined.", u.GetName()), serverUser, Filter{UserName: u.GetName()})
		r.BroadCast(msg)
		if err := r.catchup(u); err != nil {
			log.Println("unable to send catchup messages to user: %s", err)
		}
	}
}

// RemoveUser will safely remove the given user.
func (r *Room) RemoveUser(u *User) {
	r.users.Delete(u.GetName())
	msg := NewMessage(fmt.Sprintf("%s left.", u.GetName()), serverUser)
	r.BroadCast(msg)
}

// GetUsers safely returns all the users in this room.
func (r *Room) GetUsers() map[string]*User {
	users := make(map[string]*User)
	r.users.Range(func(k, val interface{}) bool {
		users[k.(string)] = val.(*User)
		return true
	})
	return users
}

// GetMessages is a safe access to the Room Message Slice
func (r *Room) GetMessages() []Message {
	r.msgsMu.Lock()
	defer r.msgsMu.Unlock()
	return r.msgs
}

// BroadCast sends the given message to all users who don't match any of the
// given filters.
func (r *Room) BroadCast(msg Message, filters ...Filter) {
	r.msgsMu.Lock()
	r.msgs = append(r.msgs, msg)
	r.msgsMu.Unlock()

	r.logger.Printf("%s: %s", r.Name, msg.String())
	r.users.Range(func(k, val interface{}) bool {
		u := val.(*User)
		if u.GetName() == msg.Author.GetName() {
			return true
		}

		if !msg.CanSend(u) {
			return true
		}

		if err := u.Send(msg.String()); err != nil {
			log.Printf("unable to send message to \"%s\": %s", u.GetName(), err)
		}

		return true
	})
}

// catchup Sends all messages to a joining user.
func (r *Room) catchup(u *User) error {
	r.msgsMu.Lock()
	defer r.msgsMu.Unlock()
	for _, msg := range r.msgs {
		if !msg.CanSend(u) {
			continue
		}

		if err := u.Send(msg.String()); err != nil {
			return fmt.Errorf("unable to send message to \"%s\": %s", u.GetName(), err)
		}
	}

	return nil
}
