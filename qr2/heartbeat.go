package qr2

import (
	"encoding/binary"
	"github.com/logrusorgru/aurora/v3"
	"net"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/logging"
)

func heartbeat(conn net.PacketConn, addr net.Addr, buffer []byte) {
	sessionId := binary.BigEndian.Uint32(buffer[1:5])
	moduleName := "QR2:" + strconv.FormatInt(int64(sessionId), 10)

	logging.Info(moduleName, "Received heartbeat from", aurora.BrightCyan(addr))
	values := strings.Split(string(buffer[5:]), "\u0000")

	payload := map[string]string{}
	for i := 0; i < len(values); i += 2 {
		if values[i] == "" {
			break
		}

		payload[values[i]] = values[i+1]
		logging.Info(moduleName, aurora.Cyan(values[i]).String()+":", aurora.Cyan(values[i+1]))
	}

	if statechanged, ok := payload["statechanged"]; ok {
		if statechanged == "1" {
			// TODO: This would be a good place to run the server->client message exploit
			// for DNS patcher games that require code patches. The status code should be
			// set to 5 at this point (if publicip is not 0), which is required.
			logging.Notice(moduleName, "Client session update")
			// Fall through
		}

		if statechanged == "2" {
			logging.Notice(moduleName, "Client session shutdown")
			removeSession(sessionId)
			return
		}
	}

	realIP, realPort := common.IPFormatToString(addr.String())

	publicIPKey, hasPublicIPKey := payload["publicip"]
	if !hasPublicIPKey || publicIPKey != realIP {
		// Set the public IP key to the real IP, and then send the challenge (done later)
		payload["publicip"] = realIP
		payload["publicport"] = realPort
	}

	session, ok := setSessionData(sessionId, payload)
	if !ok {
		return
	}

	if !session.Authenticated || !hasPublicIPKey || publicIPKey != realIP {
		logging.Notice(moduleName, "Sending challenge")
		sendChallenge(conn, addr, session)
		return
	}
}
