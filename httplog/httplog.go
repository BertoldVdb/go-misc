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

type requestObserver struct {
	http.ResponseWriter

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

	if s.code == 0 {
		s.code = 200
	}

	return n, err
}

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

// GetMiddleware returns a function that can be passed to Use to enable the logger.
// For example: mux.Use(httpLog.GetMiddleware())
func (l *HTTPLog) GetMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			begin := time.Now()

			id := ""
			if len(l.CorrelationHeader) > 0 {
				id = r.Header.Get(l.CorrelationHeader)
			}
			if len(id) == 0 {
				id = uuid.New().String()
			} else if len(id) > 40 {
				id = id[0:40]
			}
			if len(l.CorrelationHeader) > 0 {
				w.Header().Set(l.CorrelationHeader, id)
			}

			keeper := requestLogKeeper{
				corrID:  id,
				httpLog: l,
			}

			extendedCtx := context.WithValue(context.WithValue(r.Context(),
				contextCorrelationID, id),
				contextKeeper, &keeper)

			ro := requestObserver{
				ResponseWriter: w,
			}

			next.ServeHTTP(&ro, r.WithContext(extendedCtx))
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

			l.logf("{%s}: HandlerCompleted [%s \"%s %s\" %d(%s) %dbytes %s \"%s\"]%s", id, r.RemoteAddr, r.Method, r.URL.RequestURI(), ro.code, http.StatusText(ro.code), ro.bytes, duration.String(), r.UserAgent(), extraLogString)
		})
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
func LogfFromRequest(r *http.Request) func(format string, param ...interface{}) {
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

	return func(format string, param ...interface{}) {
		panic("Request has no logger attached")
	}
}
