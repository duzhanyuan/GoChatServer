TCPChat Server
==============

This is a quick little experiment in Go where I am attempting to create a quick
little chat server with usernames and perhaps a few nice features if I ever
think of what those features are going to be.

This will be TCP-based, so I should define a couple types of packets that will
be sent and construct the chat protocol on top of that. As I decide to flesh
out the chat system, the architecture and the packets themselves may change.

This is not meant to be the 'best' way of doing this nor even the most
efficient. I'm just trying to get some more practice writing network code in
Go.

## High-level Feature Description

When a user joins, they send a Reg (Register) packet with their desired
username and the chat room they wish to join. If their username is already
used, they will get an error back.  If the username is *not* already used,
then the server will associate that username with the connection's IP address.
The confirmation of this association is sent as a RegAck (Register
Acknowledgement) packet. At this point, the server will check if the chatroom
has been created. If it has, then the user will be associated with that
chatroom. If not, then the server will create that chatroom and then associate
the user. Once this has been done, the user will receive an OKSend packet
which will signify that they are allowed to start sending messages.

The client will keep a connection open to the server. Messages will stream in,
notifying the client of (message contents, message timestamp, message sent by
user). When the client wants to send a message, it will be sent as a packet of
(message contents, user). "Send" times are determined by when the server
receives the TCP packet, and thus message order will be enforced on the server.
When the server receives a message, it should check that the username is
associated with the correct IP.

When a client disconnects, it will send a DEREG (deregister) packet to the
server to notifiy the server that it is no longer associated with that
username/chatroom.

## Packets

HEADER:

+-------+---------------------+
| byte  |                     |
+-------+---------------------+
|   1   |  message type       |
+-------+---------------------+
|  2-10 |    username         |
+-------+---------------------+
| 11-20 |    chatroom         |
+-------+---------------------+

Here, we can see that the username can be at most 9 bytes long, and the
chatroom name can be at most 10 bytes long. When the client constructs
the packet, it will pack the username and chatroom segments with null
bytes so that the server can give you a username like "gabe" and not
"gabe00000".

REG  0x01 -- sent by client: (username, chatroom)

If we see a REG packet on the server, we don't look for a payload. We just make
the associations of (username, IPaddress) and (username, chatroom).

REGACK 0x02 -- sent by server: (username, chatroom)

Once we see a REGACK packet from the server, we don't look for a payload. We just
drop the user into a loop where they can start

OKSEND 0x03 -- sent by server

not implemented yet

MSGRECV 0x04 -- sent by server

+-------+---------------------+
| byte  |                     |
+-------+---------------------+
|  21   | len of message      |
+-------+---------------------+
| 22-?  |    message          |
+-------+---------------------+

MSGSEND 0x05 -- sent by client

+-------+---------------------+
| byte  |                     |
+-------+---------------------+
|  21   | len of message      |
+-------+---------------------+
| 22-?  |    message          |
+-------+---------------------+

DEREG 0x06 -- sent by client

PING 0x00 -- sent by server and client

every N seconds (probably 5?) the server sends a ping to the client
to check if it is still there. If the client does not respond, then
the client is dropped
