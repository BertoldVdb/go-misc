package httplog

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Logger is a function that can be used for logging. It has the same signature
// as log.Printf
type Logger func(string, ...interface{})

// HTTPLog is a simple logging middleware that can be added to the net/http server.
// It supports basic request correlation.
type HTTPLog struct {
	LogOut     Logger
	ServerName string

	CorrelationHeader string
}

type httpLogContextKey int

const (
	contextCorrelationID httpLogContextKey = 1
	contextKeeper        httpLogContextKey = 2
)

func (l *HTTPLog) logf(format string, param ...interface{}) {
	if l.LogOut != nil {
		var p []interface{}
		p = append(p, l.ServerName)
		p = append(p, param...)
		l.LogOut("HTTPd [%s]: "+format, p...)
	}
}

type requestObserver struct {
	http.ResponseWriter
	http.Hijacker

	bytes int
	code  int
}

func (s *requestObserver) WriteHeader(code int) {
	s.ResponseWriter.WriteHeader(code)
	s.code = code
}

func (s *requestObserver) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n

	return n, err
}

type handlerType struct {
	http.Handler

	httpLog *HTTPLog
	next    http.Handler
}

func (h *handlerType) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	begin := time.Now()

	id := ""
	if len(h.httpLog.CorrelationHeader) > 0 {
		id = r.Header.Get(h.httpLog.CorrelationHeader)
	}
	if len(id) == 0 {
		id = uuid.New().String()
	} else if len(id) > 40 {
		id = id[0:40]
	}
	if len(h.httpLog.CorrelationHeader) > 0 {
		w.Header().Set(h.httpLog.CorrelationHeader, id)
	}

	keeper := requestLogKeeper{
		corrID:  id,
		httpLog: h.httpLog,
	}

	extendedCtx := context.WithValue(context.WithValue(r.Context(),
		contextCorrelationID, id),
		contextKeeper, &keeper)

	ro := requestObserver{
		ResponseWriter: w,
		code:           200,
	}

	// Required for websocket support
	switch wt := w.(type) {
	case http.Hijacker:
		ro.Hijacker = wt
	}

	h.next.ServeHTTP(&ro, r.WithContext(extendedCtx))
	duration := time.Now().Sub(begin)

	keeper.Lock()
	keeper.done = true
	extraLog := keeper.output
	keeper.output = nil
	keeper.Unlock()

	extraLogString := ""
	if len(extraLog) > 0 {
		extraLogString = ": " + strings.Join(extraLog, ", ")
	}

	h.httpLog.logf("{%s}: HandlerCompleted [%s \"%s %s\" %d(%s) %dbytes %s \"%s\"]%s", id, r.RemoteAddr, r.Method, r.URL.RequestURI(), ro.code, http.StatusText(ro.code), ro.bytes, duration.String(), r.UserAgent(), extraLogString)
}

// GetHandler returns a function that goes in between the server and the real handler
func (l *HTTPLog) GetHandler(next http.Handler) http.Handler {
	return &handlerType{
		next:    next,
		httpLog: l,
	}
}

type requestLogKeeper struct {
	sync.Mutex

	httpLog *HTTPLog
	corrID  string
	output  []string
	done    bool
}

// CorrelationIDFromRequest returns the correlation ID associated with a http.Request
func CorrelationIDFromRequest(r *http.Request) string {
	v := r.Context().Value(contextCorrelationID)
	if v == nil {
		return "None"
	}
	return v.(string)
}

// LogfFromRequest returns a function with fmt.Printf signature that will write to the log associated
// with the request
func LogfFromRequest(r *http.Request) Logger {
	switch keeper := r.Context().Value(contextKeeper).(type) {
	case *requestLogKeeper:
		return func(format string, param ...interface{}) {
			printf := fmt.Sprintf("\""+format+"\"", param...)

			keeper.Lock()
			defer keeper.Unlock()

			if keeper.done {
				keeper.httpLog.logf("{%s}: %s", keeper.corrID, printf)
				return
			}

			keeper.output = append(keeper.output, printf)
		}
	}

	return nil
}
