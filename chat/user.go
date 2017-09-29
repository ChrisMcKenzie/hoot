package chat

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
)

var serverUser = &User{
	Name: "server",
}

const HelpMessage = `
/name <name>	- Changes name
/room [name]	- Changes room to given room and prompts to create if
			it doesn't exist. if no option is given will return the
			current room

/rooms		- Will return a list of rooms
/quit		- will disconnect from chat server
`

// User defines a person or machine who has connected via tcp interface and
// handles all state, messaging, etc for a given connection.
type User struct {
	Name   string
	NameMu sync.RWMutex

	CurrentRoom string

	conn net.Conn

	r   *bufio.Reader
	w   *bufio.Writer
	wMu sync.Mutex

	rm *RoomManager

	filters []Filter

	closeCh chan struct{}
}

// Ask the user for there name, and create a new User joining them to the
// Default room
func NewUser(conn net.Conn, rm *RoomManager) *User {
	u := &User{
		conn:    conn,
		r:       bufio.NewReader(conn),
		w:       bufio.NewWriter(conn),
		rm:      rm,
		closeCh: make(chan struct{}),
	}

	if err := u.preamble(); err != nil {
		log.Println(err)
	}

	if err := u.setup(); err != nil {
		log.Printf("unable to setup client: %s", err)
	}

	go u.handle()

	return u
}

func (u *User) preamble() error {
	if err := u.Send("Please enter your name.\n"); err != nil {
		return fmt.Errorf("unable to send to client: %s", err)
	}

	name, err := u.readLine()
	if err != nil {
		return fmt.Errorf("unable to read from client: %s", err)
	}

	u.SetName(name)

	return nil
}

func (u *User) JoinRoom(name string) error {
	if u.CurrentRoom != "" {
		u.rm.LeaveRoom(u, u.CurrentRoom)
	}

	u.CurrentRoom = name
	return u.rm.JoinRoom(u, name)
}

func (u *User) createRoom(name string) error {
	err := u.Send(fmt.Sprintf("Room %s does not exist, would you like to create it? [y/N]\n", name))
	if err != nil {
		return err
	}

	answer, err := u.readLine()
	if err != nil {
		log.Printf("unable to read from client: %s", err)
	}

	if answer == "" || strings.ContainsAny(answer, "nN") {
		return nil
	}

	u.rm.CreateRoom(name)
	return nil
}

func (u *User) handle() {
	for {
		select {
		case _, ok := <-u.closeCh:
			if !ok {
				return
			}
		default:
			content, err := u.readLine()
			if err != nil {
				if err == io.EOF {
					if err := u.Close(); err != nil {
						log.Printf("unable to close user connection: %s", err)
					}
					log.Printf("%s left", u.GetName())
					return
				}

				log.Printf("unable to read data from client: %s", err)
			}

			if content == "" {
				if err := u.drawPrompt(); err != nil {
					log.Printf("unable to draw prompt for client: %s", err)
				}
				continue
			}

			if strings.HasPrefix(content, "/") {
				command := strings.TrimPrefix(content, "/")
				if err := u.parseCommand(command); err != nil {
					log.Printf("unable to parse user command: %s", err)
				}

				continue
			}

			msg := NewMessage(content, u)

			u.rm.Broadcast(u.CurrentRoom, msg)
			log.Printf("%s sent message", u.GetName())

			if err := u.drawPrompt(); err != nil {
				log.Printf("unable to draw prompt for client: %s", err)
			}
		}
	}
}

func (u *User) setup() error {
	if err := u.clearAndSend("\r\033[1;25r"); err != nil {
		return fmt.Errorf("unable to setup client: %s", err)
	}

	return u.drawPrompt()
}

func (u *User) Close() error {
	close(u.closeCh)
	u.rm.LeaveRoom(u, u.CurrentRoom)
	return u.conn.Close()
}

func (u *User) parseCommand(command string) error {
	if len(command) < 0 {
		return fmt.Errorf("expected command after \"/\"")
	}

	parts := strings.Split(command, " ")
	action := parts[0]

	switch action {
	case "name":
		if len(parts[1]) < 0 {
			return fmt.Errorf("expected <name> argument after action")
		}
		if err := u.SetName(parts[1]); err != nil {
			return err
		}
		if err := u.drawPrompt(); err != nil {
			log.Printf("unable to draw prompt for client: %s", err)
		}
	case "ignore":
		if len(parts) <= 1 {
			return fmt.Errorf("expected <name> argument after action")
		}
		u.filters = append(u.filters, Filter{UserName: parts[1]})

		if err := u.drawPrompt(); err != nil {
			log.Printf("unable to draw prompt for client: %s", err)
		}
	case "room":
		if len(parts) == 1 {
			if err := u.Send(fmt.Sprintf("you are currently in \"%s\"", u.CurrentRoom)); err != nil {
				return err
			}
			return nil
		}
		if !u.rm.RoomExists(parts[1]) {
			if err := u.createRoom(parts[1]); err != nil {
				return err
			}
		}

		if err := u.JoinRoom(parts[1]); err != nil {
			return err
		}

		if err := u.drawPrompt(); err != nil {
			log.Printf("unable to draw prompt for client: %s", err)
		}
	case "rooms":
		rooms := u.rm.GetRooms()
		for name, _ := range rooms {
			if err := u.Send(name); err != nil {
				return err
			}
		}
		if err := u.drawPrompt(); err != nil {
			log.Printf("unable to draw prompt for client: %s", err)
		}
	case "quit":
		if err := u.Close(); err != nil {
			return err
		}
		log.Printf("%s left", u.GetName())
	case "help":
		if err := u.Send(HelpMessage); err != nil {
			return err
		}
		if err := u.drawPrompt(); err != nil {
			log.Printf("unable to draw prompt for client: %s", err)
		}
	}

	return nil
}

func (u *User) GetName() string {
	u.NameMu.RLock()
	defer u.NameMu.RUnlock()

	return u.Name
}

func (u *User) SetName(name string) error {
	u.NameMu.Lock()
	u.Name = name
	u.NameMu.Unlock()

	u.rm.Broadcast(u.CurrentRoom,
		NewMessage(fmt.Sprintf("changed name to %s", name), u))
	return u.Send("name changed.")
}

func (u *User) Send(m string) error {
	_, err := u.w.WriteString(fmt.Sprintf("\033[1L\033[0G%s\n\033[3G", m))
	if err != nil {
		return err
	}

	return u.drawPrompt()
}

func (u *User) drawPrompt() error {
	_, err := u.w.WriteString("\r\033[0G> ")
	if err != nil {
		return err
	}

	return u.w.Flush()
}

func (u *User) clear() error {
	return u.Send("\033[2J")
}

func (u *User) clearAndSend(m string) error {
	return u.Send(fmt.Sprintf("\033[H\033[2J%s", m))
}

func (u *User) readLine() (string, error) {
	byts, prefix, err := u.r.ReadLine()
	// check if line is longer than buffer, if so read next "line" for the rest
	if prefix {
		suf, _, err := u.r.ReadLine()
		if err != nil {
			return "", err
		}

		byts = append(byts, suf...)
	}
	msg := string(byts)

	if err != nil {
		return msg, err
	}

	return msg, nil
}
