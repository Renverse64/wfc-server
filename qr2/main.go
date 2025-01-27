package qr2

import (
	"encoding/binary"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strconv"
	"sync"
	"time"
	"wwfc/common"
	"wwfc/logging"
)

var (
	// I would use a sync.Map instead of the map mutex combo, but this performs better.
	sessions   = map[uint32]*Session{}
	mutex      = sync.RWMutex{}
	masterConn net.PacketConn
)

const (
	QueryRequest            = 0x00
	ChallengeRequest        = 0x01
	EchoRequest             = 0x02
	HeartbeatRequest        = 0x03
	AddErrorRequest         = 0x04
	EchoResponseRequest     = 0x05
	ClientMessageRequest    = 0x06
	ClientMessageAckRequest = 0x07
	KeepAliveRequest        = 0x08
	AvailableRequest        = 0x09
	ClientRegisteredReply   = 0x0A
)

func StartServer() {
	// Get config
	config := common.GetConfig()

	address := config.Address + ":27900"
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		panic(err)
	}

	masterConn = conn

	// Close the listener when the application closes.
	defer conn.Close()
	logging.Notice("QR2", "Listening on", address)

	for {
		buf := make([]byte, 1024)
		_, addr, err := conn.ReadFrom(buf)
		if err != nil {
			continue
		}

		go handleConnection(conn, addr, buf)
	}
}

func handleConnection(conn net.PacketConn, addr net.Addr, buffer []byte) {
	if buffer[0] == AvailableRequest {
		logging.Notice("QR2", "Command:", aurora.Yellow("AVAILABLE"))
		conn.WriteTo(createResponseHeader(AvailableRequest, 0), addr)
		return
	}

	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	moduleName := "QR2:" + strconv.FormatInt(int64(sessionId), 10)

	switch buffer[0] {
	case QueryRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("QUERY"))
		break

	case ChallengeRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("CHALLENGE"))

		mutex.Lock()
		if sessions[sessionId].Challenge != "" {
			// TODO: Verify the challenge
			sessions[sessionId].Authenticated = true
			mutex.Unlock()

			conn.WriteTo(createResponseHeader(ClientRegisteredReply, sessionId), addr)
		} else {
			mutex.Unlock()
		}
		break

	case EchoRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("ECHO"))
		break

	case HeartbeatRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("HEARTBEAT"))
		heartbeat(conn, addr, buffer)
		break

	case AddErrorRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("ADDERROR"))
		break

	case EchoResponseRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("ECHO_RESPONSE"))
		break

	case ClientMessageRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("CLIENT_MESSAGE"))
		return

	case ClientMessageAckRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("CLIENT_MESSAGE_ACK"))
		return

	case KeepAliveRequest:
		logging.Notice(moduleName, "Command:", aurora.Yellow("KEEPALIVE"))
		sessionId := binary.BigEndian.Uint32(buffer[1:5])
		mutex.Lock()
		sessions[sessionId].LastKeepAlive = time.Now().Unix()
		mutex.Unlock()
		return

	case AvailableRequest:
		return

	case ClientRegisteredReply:
		logging.Notice(moduleName, "Command:", aurora.Cyan("CLIENT_REGISTERED"))
		break

	default:
		logging.Error(moduleName, "Unknown command:", aurora.Yellow(buffer[0]))
		return
	}
}

func createResponseHeader(command byte, sessionId uint32) []byte {
	return binary.BigEndian.AppendUint32([]byte{0xfe, 0xfd, command}, sessionId)
}
