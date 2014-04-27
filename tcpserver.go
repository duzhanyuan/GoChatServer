package main

import (
  "net"
  "os"
  "fmt"
  "errors"
  "time"
)

const (
  RECV_BUF_LEN = 1024
)

var Clients = make(map[string]*client)
var Members = make(map[string][](*client))
var Chatrooms = make(map[string]chan *msg)

func Log(v ...interface{}) {
  fmt.Println(v...)
}

type client struct {
  user string
  chatroom string
  addr string
  conn net.Conn
}

type msg struct {
  user string
  chatroom string
  message string
  timestamp time.Time
}

func pingclient(c *client) {
  for {
    timer := time.NewTimer(5 * time.Second)
    <- timer.C
    packet := constructHeader(0, c.user, c.chatroom)
    _, err := c.conn.Write(packet)
    if err != nil {
      Log("disconnect client", c)
      Clients[c.user] = nil
      return
    }
  }
}

func distributeMessages(chatroom string) {
  for {
    m := <- Chatrooms[chatroom]
    Log("distributing message", m.message)
    for _, c := range Members[chatroom] {
      Log("send",m.message,"to",c.user)
      packet := constructMessage(m.user, chatroom, []byte(m.message))
      c.conn.Write(packet)
    }
  }
}

func constructMessage(user, chatroom string, message []byte) []byte {
  packet := constructHeader(4, user, chatroom)
  packet = append(packet, byte(len(message)))
  packet = append(packet, message...)
  return packet
}

func constructHeader(msgtype int, user, chatroom string) []byte {
  packet := [20]byte{}
  packet[0] = byte(msgtype)
  userbyte := []byte(user)
  for i, b := range(userbyte) {
    packet[1+i] = b
  }
  chatbyte := []byte(chatroom)
  for i, c := range(chatbyte) {
    packet[12+i] = c
  }
  return packet[:]
}

func parsePacket(packet []byte) (int, string, string, []byte, error) {
  if len(packet) < 20 {
    fmt.Println("Packet too short. Has length", len(packet))
    return  0, "", "", []byte{}, errors.New("Invalid packet")
  }

  msgtype := int(packet[0])
  user := string(packet[1:11])
  chatroom := string(packet[12:20])

  Log("Received message with msgtype", msgtype, "user", user, "chatroom", chatroom)
  return msgtype, user, chatroom, packet[20:], nil
}

func handleRegister(user, chatroom string, addr string) *client {
  // attempts to register (user, addr) and (user, chatroom).
  // returns true if registration successful, else false
  Log("Attempt to register",user,chatroom,addr)
  if Clients[user] != nil {
    Log("Already user with name", user)
    return nil
  }
  if Chatrooms[chatroom] == nil {
    Chatrooms[chatroom] = make(chan *msg)
    go distributeMessages(chatroom)
  }
  Log("Successfully registered",user,chatroom,addr)
  c := &client{user: user, chatroom: chatroom, addr: addr}
  Clients[user] = c
  Members[chatroom] = append(Members[chatroom], c)
  return c
}

func sendREGACK(user, chatroom string, conn net.Conn) {
  header := constructHeader(2, user, chatroom)
  conn.Write(header)
}

func handleSentMessage(user, chatroom string, rest []byte , addr string) *msg {
  if Clients[user] != nil && Clients[user].addr != addr {
    Log("Client",Clients[user],"does not match IPAddress")
    return nil
  }

  msglen := int(rest[0])
  message := string(rest[1:1+msglen])
  m := &msg{user, chatroom, message, time.Now()}
  Log("got message",message,"of length",msglen,"from",user,"to room",chatroom)
  return m
}

func handleConnection(conn net.Conn) {
  // extract IP address of connection
  ipaddr := conn.RemoteAddr().String()
  for {
    // read in RECV_BUF_LEN bytes
    buf := make([]byte, RECV_BUF_LEN)
    n, err := conn.Read(buf)
    if err != nil {
      return
    }

    Log("received ", n, " bytes of data =", buf[:n])

    // parse the header. This should eventually return the rest of the buffer
    // along with the msgtype, user and chatroom
    msgtype, user, chatroom, rest, err := parsePacket(buf[:n])
    if err != nil {
      Log("Invalid packet, closing conn")
      conn.Close()
    }

    // switch statement to handle different types of packet
    switch {
      // REG packet from user
      case msgtype == 1:
        c := handleRegister(user, chatroom, ipaddr)
        if c != nil {
          c.conn = conn
          timer := time.NewTimer(1 * time.Second) // sleep 1 second just to be difficult
          <- timer.C
          sendREGACK(user, chatroom, conn)
          go pingclient(Clients[user])
        }
      // SENDMSG packet from user
      case msgtype == 5:
        m := handleSentMessage(user, chatroom, rest, ipaddr)
        if m != nil{
          Chatrooms[chatroom] <- m
        }


        Log("sendmsg received from", ipaddr)
      case msgtype == 6:
        Log("deregister received from", ipaddr)
    }
  }
}

func main() {
  // parse host, port from command line args
  if len(os.Args) != 3 {
    fmt.Println("tcpserver <host> <port>")
    os.Exit(1)
  }
  host := os.Args[1]
  port := os.Args[2]

  // configure and create TCP server
  service := host + ":" + port
  Log("listening on "+service)
  listener, err := net.Listen("tcp", service)
  if err != nil {
    Log("Error creating TCP server: ", err.Error())
    os.Exit(1)
  }

  // server listen loop
  for {
    conn, err := listener.Accept()
    if err != nil {
      Log("Error with connection: ", err.Error())
      continue
    }
    go handleConnection(conn)
  }

}
