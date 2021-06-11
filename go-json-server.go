package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/tkc/go-json-server/src/logger"
)

const (
	charsetUTF8 = "charset=UTF-8"
	CONV_TOKEN  = "CONV-0123456789"
)

const (
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMETextXML                          = "text/xml"
	MIMETextXMLCharsetUTF8               = MIMETextXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

const (
	HeaderAccept                        = "Accept"
	HeaderAcceptEncoding                = "Accept-Encoding"
	HeaderAllow                         = "Allow"
	HeaderAuthorization                 = "Authorization"
	HeaderContentDisposition            = "Content-Disposition"
	HeaderContentEncoding               = "Content-Encoding"
	HeaderContentLength                 = "Content-Length"
	HeaderContentType                   = "Content-Type"
	HeaderConversationToken             = "Conversation-Token"
	HeaderCookie                        = "Cookie"
	HeaderSetCookie                     = "Set-Cookie"
	HeaderIfModifiedSince               = "If-Modified-Since"
	HeaderLastModified                  = "Last-Modified"
	HeaderLocation                      = "Location"
	HeaderUpgrade                       = "Upgrade"
	HeaderVary                          = "Vary"
	HeaderWWWAuthenticate               = "WWW-Authenticate"
	HeaderXForwardedFor                 = "X-Forwarded-For"
	HeaderXForwardedProto               = "X-Forwarded-Proto"
	HeaderXForwardedProtocol            = "X-Forwarded-Protocol"
	HeaderXForwardedSsl                 = "X-Forwarded-Ssl"
	HeaderXUrlScheme                    = "X-Url-Scheme"
	HeaderXHTTPMethodOverride           = "X-HTTP-Method-Override"
	HeaderXRealIP                       = "X-Real-IP"
	HeaderXRequestID                    = "X-Request-ID"
	HeaderServer                        = "Server"
	HeaderOrigin                        = "Origin"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"
	HeaderStrictTransportSecurity       = "Strict-Transport-Security"
	HeaderXContentTypeOptions           = "X-Content-Type-Options"
	HeaderXXSSProtection                = "X-XSS-Protection"
	HeaderXFrameOptions                 = "X-Frame-Options"
	HeaderContentSecurityPolicy         = "Content-Security-Policy"
	HeaderXCSRFToken                    = "X-CSRF-Token"
)

type Endpoint struct {
	Type    string   `json:"type"`
	Methods []string `json:"methods"`
	Status  int      `json:"status"`
	Path    string   `json:"path"`
	Folder  string   `json:"folder"`
}

type API struct {
	Host      string     `json:"host"`
	Port      int        `json:"port"`
	Endpoints []Endpoint `json:"endpoints"`
}

type RequestBody struct {
	Query string `json:"query"`
}

var api API

var base_dir string

func main() {

	argLength := len(os.Args[1:])
	if argLength != 1 {
		base_dir = "."
	} else {
		base_dir = os.Args[1]
	}

	raw, err := ioutil.ReadFile(base_dir + "/api.json")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	json.Unmarshal(raw, &api)
	if err != nil {
		log.Fatal(" ", err)
	}

	for _, ep := range api.Endpoints {
		log.Print(ep)
		if len(ep.Folder) > 0 {
			http.Handle(ep.Path+"/", http.StripPrefix(ep.Path+"/", http.FileServer(http.Dir(ep.Folder))))
		} else {
			http.HandleFunc(ep.Path, response)
		}
	}

	err = http.ListenAndServe(":"+strconv.Itoa(api.Port), nil)

	if err != nil {
		log.Fatal(" ", err)
	}
}

func response(w http.ResponseWriter, r *http.Request) {

	appLogger := logger.CreateLogger()

	r.ParseForm()
	appLogger.AccessLog(r)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers",
		"Origin, X-Requested-With, Content-Type, Accept, Authorization, Conversation-Token")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	conversation_header := w.Header().Get(HeaderConversationToken)
	reqBody, err := getRequestBody(*r)
	if err != nil {
		log.Fatal(" ", err)
		os.Exit(1)
	}

	for _, ep := range api.Endpoints {
		// check if method matches
		methodMatches := false
		for _, m := range ep.Methods {
			if m == r.Method {
				methodMatches = true
				break
			}
		}
		if r.URL.Path == ep.Path && methodMatches {
			w.Header().Set(HeaderContentType, MIMEApplicationJSON)
			if conversation_header == "" {
				w.Header().Set(HeaderConversationToken, CONV_TOKEN)
			} else {
				w.Header().Set(HeaderConversationToken, conversation_header)
			}
			w.WriteHeader(ep.Status)
			s := path2Response(ep.Path, reqBody.Query)
			b := []byte(s)
			w.Write(b)
		}
		continue
	}
}

func path2Response(path string, query string) string {

	file, err := os.Open(base_dir + path + ".json")
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer file.Close()
	buf := new(bytes.Buffer)
	buf.ReadFrom(file)
	if query != "" {
		fmt.Println("Recieved Query: ", query)
	}
	return buf.String()
}

func getRequestBody(r http.Request) (RequestBody, error) {
	var b RequestBody
	err := json.NewDecoder(r.Body).Decode(&b)
	if err != nil {
		return b, err
	}
	return b, nil
}
