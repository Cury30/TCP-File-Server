package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

//Command Constants
const (
	CMD_SUBSCRIBE int = iota
	CMD_SEND
	CMD_QUIT
)

// byte char delimiter
var delimiter []byte = []byte("}")

//Structs needed
type command struct {
	id      int
	client  *client
	args    []string
	content []byte
}

type channel struct {
	name    string
	members map[net.Addr]*client
}

type server struct {
	channels map[string]*channel
	commands chan command
}

type client struct {
	conn     net.Conn
	nick     string
	channel  *channel
	commands chan<- command
}

//Functions
func (r *channel) broadcast(sender *client, msg []byte) {
	msg = append(msg, delimiter...)
	for addr, m := range r.members {
		if sender.conn.RemoteAddr() != addr {
			m.msg(msg)
		}
	}
}

func newServer() *server {
	return &server{
		channels: make(map[string]*channel),
		commands: make(chan command),
	}
}

func (s *server) run() {
	for cmd := range s.commands {
		switch cmd.id {
		case CMD_SUBSCRIBE:
			s.subscribe(cmd.client, cmd.args)
		case CMD_SEND:
			s.sendFile(cmd.client, cmd.args, cmd.content)
		case CMD_QUIT:
			s.quit(cmd.client)
		}
	}
}

func (s *server) newClient(conn net.Conn) *client {
	log.Printf("new client has joined: %s", conn.RemoteAddr().String())

	return &client{
		conn:     conn,
		nick:     fmt.Sprintf("Client-%s", conn.RemoteAddr().String()),
		commands: s.commands,
	}
}

func (s *server) subscribe(c *client, args []string) {
	if len(args) < 2 {
		c.msg([]byte("channel name is required. usage: subscribe Channel#}"))
		return
	}

	channelName := strings.Trim(args[1], "}")
	r, ok := s.channels[channelName]
	if !ok {
		r = &channel{
			name:    channelName,
			members: make(map[net.Addr]*client),
		}
		s.channels[channelName] = r
	}

	r.members[c.conn.RemoteAddr()] = c

	s.quitCurrentChannel(c)
	c.channel = r

	r.broadcast(c, []byte(fmt.Sprintf("%s joined the channel", c.nick)))

	c.msg([]byte(fmt.Sprintf("welcome to %s}", channelName)))
}

func (s *server) sendFile(c *client, args []string, content []byte) {
	if len(args) < 3 {
		c.msg([]byte("Wrong sintax, usage example: send fileName channel#}"))
		return
	}
	var subArgs []string
	subArgs = append(subArgs, "subscribe")
	subArgs = append(subArgs, args[2])

	s.subscribe(c, subArgs)
	c.channel.broadcast(c, content)
	s.quitCurrentChannel(c)
}

func (s *server) quit(c *client) {
	log.Printf("client has left the channel: %s", c.conn.RemoteAddr().String())

	s.quitCurrentChannel(c)

	c.msg([]byte("sad to see you go =(}"))
	c.conn.Close()
}

func (s *server) quitCurrentChannel(c *client) {
	if c.channel != nil {
		oldChannel := s.channels[c.channel.name]
		delete(s.channels[c.channel.name].members, c.conn.RemoteAddr())
		oldChannel.broadcast(c, []byte(fmt.Sprintf("%s has left the channel", c.nick)))
	}
}

func (c *client) readInput() {
	for {
		myReader := bufio.NewReader(c.conn)
		myMsg, err := myReader.ReadBytes(byte(125))
		if err != nil {
			return
		}

		splittedMsg := strings.Split(string(myMsg), " ")

		switch splittedMsg[0] {
		case "subscribe":
			c.commands <- command{
				id:     CMD_SUBSCRIBE,
				client: c,
				args:   splittedMsg,
			}
		case "send":
			contentLength, err := strconv.Atoi(strings.TrimRight(splittedMsg[3], "}"))
			if err != nil {
				log.Printf("Error reading the file size: %s", err.Error())
				return
			}

			content := make([]byte, contentLength)
			j, err := io.ReadFull(myReader, content)
			if err != nil {
				log.Printf("Error reading the buffer: %s, bytes: %d", err.Error(), j)
				return
			}

			myMsg = append(myMsg, content[:]...)

			c.commands <- command{
				id:      CMD_SEND,
				client:  c,
				args:    splittedMsg[:3],
				content: myMsg,
			}
		case "receive":
			c.commands <- command{
				id:     CMD_SUBSCRIBE,
				client: c,
				args:   splittedMsg,
			}
		default:
			c.err(fmt.Errorf("unknown command: %s", strings.TrimSpace(splittedMsg[0])))
		}
	}
}

func (c *client) err(err error) {
	c.conn.Write([]byte("err: " + err.Error() + "}"))
}

func (c *client) msg(msg []byte) {
	c.conn.Write(msg)
}
