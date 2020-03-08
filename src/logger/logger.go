package logger

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	stdlog "log"
	"net/http"
	"os"
	"strconv"
)

type stdLogger struct {
	stderr *stdlog.Logger
	stdout *stdlog.Logger
}

func CreateLogger() *stdLogger {
	return &stdLogger{
		stdout: stdlog.New(os.Stdout, "", 0),
		stderr: stdlog.New(os.Stderr, "", 0),
	}
}

func (l *stdLogger) AccessLog(r *http.Request) {
	file, err := os.OpenFile("log.csv", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()

	jsonBody := dumpJsonBoddy(r)

	s := []string{r.Method, r.Host, r.Proto, r.RequestURI, r.RemoteAddr, jsonBody}
	writer := csv.NewWriter(file)
	writer.Write(s)
	writer.Flush()
}

func dumpJsonBoddy(req *http.Request) string {

	if req.Method == "GET" {
		return ""
	}

	if req.Header.Get("Content-Type") != "application/json" {
		return ""
	}

	length, err := strconv.Atoi(req.Header.Get("Content-Length"))

	if err != nil {
		return ""
	}

	body := make([]byte, length)
	length, err = req.Body.Read(body)

	if err != nil && err != io.EOF {
		return ""
	}

	var jsonBody map[string]interface{}
	err = json.Unmarshal(body[:length], &jsonBody)

	if err != nil {
		return ""
	}

	s := fmt.Sprintf("%v", jsonBody)
	return s
}

func (l *stdLogger) Printf(format string, args ...interface{}) {
	l.stdout.Printf(format, args...)
}

func (l *stdLogger) Errorf(format string, args ...interface{}) {
	l.stderr.Printf(format, args...)
}

func (l *stdLogger) Fatalf(format string, args ...interface{}) {
	l.stderr.Fatalf(format, args...)
}
