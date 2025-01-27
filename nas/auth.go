package nas

import (
	"github.com/logrusorgru/aurora/v3"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"wwfc/common"
	"wwfc/database"
	"wwfc/logging"
)

func handleAuthRequest(moduleName string, w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		logging.Error(moduleName, "Failed to parse form")
		replyHTTPError(w, 400, "400 Bad Request")
		return
	}

	fields := map[string]string{}
	for key, values := range r.PostForm {
		if len(values) != 1 {
			logging.Warn(moduleName, "Ignoring multiple POST form values:", aurora.Cyan(key).String()+":", aurora.Cyan(values))
			continue
		}

		parsed, err := common.Base64DwcEncoding.DecodeString(values[0])
		if err != nil {
			logging.Error(moduleName, "Invalid POST form value:", aurora.Cyan(key).String()+":", aurora.Cyan(values[0]))
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}
		logging.Info(moduleName, aurora.Cyan(key).String()+":", aurora.Cyan(string(parsed)))
		fields[key] = string(parsed)
	}

	reply := map[string]string{}

	if r.URL.String() == "/ac" {
		action, ok := fields["action"]
		if !ok || action == "" {
			logging.Error(moduleName, "No action in form")
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}

		switch action {
		case "acctcreate":
			reply = acctcreate(moduleName, fields)
			break

		case "login":
			reply = login(moduleName, fields)
			break

		case "svcloc":
			reply = svcloc(moduleName, fields)
			break

		default:
			logging.Error(moduleName, "Unknown action:", aurora.Cyan(action))
			reply = map[string]string{
				"retry":    "0",
				"returncd": "109",
			}
			break
		}
	} else if r.URL.String() == "/pr" {
		words, ok := fields["words"]
		if words == "" || !ok {
			logging.Error(moduleName, "No words in form")
			replyHTTPError(w, 400, "400 Bad Request")
			return
		}

		reply = handleProfanity(moduleName, fields)
	}

	param := url.Values{}
	for key, value := range reply {
		param.Set(key, common.Base64DwcEncoding.EncodeToString([]byte(value)))
	}
	response := []byte(param.Encode())
	response = []byte(strings.Replace(string(response), "%2A", "*", -1))
	// DWC treats the response like a null terminated string
	response = append(response, 0x00)

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(response)))
	w.Write(response)
}

func acctcreate(moduleName string, fields map[string]string) map[string]string {
	return map[string]string{
		"retry":    "0",
		"returncd": "002",
		"userid":   strconv.FormatInt(database.GetUniqueUserID(), 10),
	}
}

func login(moduleName string, fields map[string]string) map[string]string {
	param := map[string]string{
		"retry":   "0",
		"locator": "gs.wiilink24.com",
	}

	strUserId, ok := fields["userid"]
	if !ok {
		logging.Error(moduleName, "No userid in form")
		param["returncd"] = "103"
		return param
	}

	userId, err := strconv.ParseInt(strUserId, 10, 64)
	if err != nil {
		logging.Error(moduleName, "Invalid userid string in form")
		param["returncd"] = "103"
		return param
	}

	gsbrcd, ok := fields["gsbrcd"]
	if !ok {
		logging.Error(moduleName, "No gsbrcd in form")
		param["returncd"] = "103"
		return param
	}

	authToken, challenge := database.GenerateAuthToken(pool, ctx, userId, gsbrcd)

	param["returncd"] = "001"
	param["challenge"] = challenge
	param["token"] = authToken
	return param
}

func svcloc(moduleName string, fields map[string]string) map[string]string {
	param := map[string]string{
		"retry":      "0",
		"returncd":   "007",
		"statusdata": "Y",
		"svchost":    "n/a",
	}

	strUserId, ok := fields["userid"]
	if !ok {
		logging.Error(moduleName, "No userid in form")
		param["returncd"] = "103"
		return param
	}

	userId, err := strconv.ParseInt(strUserId, 10, 64)
	if err != nil {
		logging.Error(moduleName, "Invalid userid string in form")
		param["returncd"] = "103"
		return param
	}

	authToken, _ := database.GenerateAuthToken(pool, ctx, userId, "")

	param["servicetoken"] = authToken
	return param
}

func handleProfanity(moduleName string, fields map[string]string) map[string]string {
	prwords := ""
	wordCount := strings.Count(fields["words"], "\t") + 1
	for i := 0; i < wordCount; i++ {
		prwords += "0"
	}

	return map[string]string{
		"returncd": "000",
		"prwords":  prwords,
	}
}
