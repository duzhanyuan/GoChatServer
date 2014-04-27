package main

import (
  "net"
  "os"
  "fmt"
  "bufio"
  "errors"
)

const (
  RECV_BUF_LEN = 1024
)

func Log(v ...interface{}) {
  fmt.Println(v...)
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

func constructMessage(user, chatroom string, message []byte) []byte {
  packet := constructHeader(5, user, chatroom)
  packet = append(packet, byte(len(message)))
  packet = append(packet, message...)
  return packet
}

func parsePacket(packet []byte) (int, string, string, []byte, error) {
  if len(packet) < 20 {
    fmt.Println("Packet too short. Has length", len(packet))
    return  0, "", "", []byte{}, errors.New("Invalid packet")
  }
    
  msgtype := int(packet[0])
  user := string(packet[1:11])
  chatroom := string(packet[12:20])

  return msgtype, user, chatroom, packet[20:], nil
}

func listenForSend(user, chatroom string, conn net.Conn) {
  for {
    bio := bufio.NewReader(os.Stdin)
    line, hasMoreInLine, err := bio.ReadLine()
    if !hasMoreInLine && err == nil{
      packet := constructMessage(user,chatroom, line)
      conn.Write(packet)
    }
  }

}

func listenForRecv(chatroom string, conn net.Conn) {
  for {
    buf := make([]byte, RECV_BUF_LEN)
    n, err := conn.Read(buf)
    if err != nil {
      return
    }
    msgtype, user, chatroom, msg, err := parsePacket(buf[:n])
    if msgtype == 4 {
      msglen := int(msg[0])
      message := string(msg[1:1+msglen])
      fmt.Println("["+user+"@"+chatroom+"]: "+message)
    }
  }
}

func main() {

  quit := make(chan bool)

  if len(os.Args) != 5 {
    fmt.Println("chatclient <host> <port> <username> <chatroom>")
    os.Exit(1)
  }
  host := os.Args[1]
  port := os.Args[2]
  username := os.Args[3]
  chatroom := os.Args[4]
  service := host + ":" + port

  conn, err := net.Dial("tcp", service)
  if err != nil {
    fmt.Println("Error connecting to tcp server: ", err.Error())
    os.Exit(1)
  }


  // test packet
  header := constructHeader(1,username,chatroom)

  _, err = conn.Write(header)
  if err != nil {
    fmt.Println("Error sending ping to tcp server: ", err.Error())
    os.Exit(1)
  }

  // main loop
  buf := make([]byte, RECV_BUF_LEN)
  n, err := conn.Read(buf)
  if err != nil {
    return
  }
  msgtype, user, chatroom, _, err := parsePacket(buf[:n])
 
  if err != nil {
    Log("Invalid packet, closing conn")
    conn.Close()
  }
  if msgtype == 2 {
    Log("Received REGACK from server. You can now chat!")
  } else {
    Log("Received",msgtype,"from server. Wasn't a REGACK, so we shut down")
    os.Exit(1)
  }

  //TODO: put "listen for messages" and "listen for sendings" into 2 go routines
  go listenForSend(user, chatroom, conn)
  go listenForRecv(chatroom, conn)

  <-quit

  Log("completed")
}
