package multirunhttp

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/BertoldVdb/go-misc/httplog"
	"github.com/BertoldVdb/go-misc/multirun"
	"github.com/sirupsen/logrus"
)

type MultiRunHTTP struct {
	multirun.Runnable

	Server     *http.Server
	LoggerHTTP *logrus.Entry
	ListenPort int
}

func (s *MultiRunHTTP) Run() error {
	if s.LoggerHTTP != nil {
		traceHTTP := httplog.HTTPLog{
			LogOut:            s.LoggerHTTP.Debugf,
			CorrelationHeader: "X-Request-ID",
			SkipInfo:          true,
		}

		go func() {
			for i := 1; i <= 10; i++ {
				time.Sleep(1 * time.Second)
				s.LoggerHTTP.Warnf("Visit http://127.0.0.1:%d/ to see the output of this program (%d/10)", s.ListenPort, i)
			}
		}()
		s.Server.Handler = traceHTTP.GetHandler(http.DefaultServeMux)
	}

	s.Server.Addr = fmt.Sprintf(":%d", s.ListenPort)
	return s.Server.ListenAndServe()
}

func (s *MultiRunHTTP) Close() error {
	return s.Server.Shutdown(context.Background())

}
