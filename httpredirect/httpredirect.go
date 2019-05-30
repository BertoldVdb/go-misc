package httpredirect

import (
	"encoding/base64"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/ocsp"
)

// Logger is a function interface that can be used for logging
type Logger func(string, ...interface{})

// RedirectServer is a simple HTTP server that is used for serving captive portal or https redirects
type RedirectServer struct {
	server             *http.Server
	Status             int
	Destination        string
	IncludeRequest     bool
	BlockOCSP          bool
	FakeInternetAccess bool
	Logger             Logger
}

func (s *RedirectServer) handleRedirectHTTPS() http.HandlerFunc {
	doRedir := func(w http.ResponseWriter, r *http.Request, destination string, req string) {
		if s.IncludeRequest {
			destination += req
		}

		if s.Logger != nil {
			s.Logger("HTTPRedirect: %s: Redirecting to %s", r.RemoteAddr, destination)
		}

		if s.Status >= 0 {
			http.Redirect(w, r, destination, s.Status)
		} else {
			page := []byte("" +
				"<html><head>" +
				"<meta http-equiv=\"refresh\" content=\"0;url=" + destination + "\"/>" +
				"<script>window.location.href = \"" + destination + "\";</script>" +
				"<title>Redirect</title></head><body>" +
				"<h1><a href=\"" + destination + "\">Click here to continue</a></h1>" +
				"</body></html>")
			w.Write(page)
		}
	}

	doOCSP := func(w http.ResponseWriter, r *http.Request, queryBytes []byte) bool {
		result, err := ocsp.ParseRequest(queryBytes)
		if err != nil {
			return false
		}

		if s.Logger != nil {
			s.Logger("HTTPRedirect: %s: Blocked %s OCSP with serial number %s", r.RemoteAddr, r.Method, result.SerialNumber)
		}

		/* DER encoded TryLater error:
		 * Sequence, length 3:
		 *   Enum, length 1:
		 *     3 (tryLater)
		 */
		w.Header().Set("Content-Type", "application/ocsp-response")
		w.Write([]byte{0x30, 0x03, 0x0a, 0x01, 0x03})
		return true
	}

	return func(w http.ResponseWriter, r *http.Request) {
		//Don't keep connection open
		w.Header().Set("Connection", "close")

		if s.Logger != nil {
			s.Logger("HTTPRedirect: %s: Processing %s request for http://%s%s", r.RemoteAddr, r.Method, r.Host, r.URL)
		}

		if s.FakeInternetAccess {
			if r.URL.Path == "/generate_204" || r.URL.Path == "/gen_204" {
				/* To fake internet access for android it is required to let https requests to https://www.google.com/ through */
				w.WriteHeader(http.StatusNoContent)
				return
			} else if r.URL.Path == "/hotspot-detect.html" || r.URL.Path == "/library/test/success.html" {
				w.Write([]byte("<HTML><HEAD><TITLE>Success</TITLE></HEAD><BODY>Success</BODY></HTML>"))
				return
			} else if r.Host == "www.msftncsi.com" {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("Microsoft NCSI"))
				return
			} else if r.Host == "www.msftconnecttest.com" {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte("Microsoft Connect Test"))
				return
			}
		}

		/*Some devices don't like to see random data when they do an OCSP query on a captive portal so we support
		 *sending a server error back */
		if s.BlockOCSP {
			//The payload can either be in a GET or POST request
			if r.Method == "POST" {
				if strings.ToLower(r.Header.Get("Content-Type")) == "application/ocsp-request" {
					queryBytes, err := ioutil.ReadAll(r.Body)
					if err == nil {
						if doOCSP(w, r, queryBytes) {
							return
						}
					}
				}
			} else if r.Method == "GET" {
				slashIndex := strings.LastIndex(r.RequestURI, "/")
				if slashIndex >= 0 {
					query, err := url.QueryUnescape(r.RequestURI[slashIndex+1:])
					if err == nil {
						queryBytes, err := base64.StdEncoding.DecodeString(query)
						if err == nil {
							if doOCSP(w, r, queryBytes) {
								return
							}
						}
					}
				}
			}
		}

		//Do redirect
		if s.Destination == "" {
			host := r.Host

			if colonIndex := strings.LastIndex(host, "]:"); colonIndex >= 0 {
				host = host[:colonIndex+1]
			} else if colonIndex := strings.LastIndex(host, ":"); colonIndex >= 0 {
				host = host[:colonIndex]
			}

			doRedir(w, r, "https://"+host, r.RequestURI)
			return
		}

		doRedir(w, r, s.Destination, r.RequestURI)
	}
}

// NewHTTPRedirect creates a new RedirectServer
func NewHTTPRedirect() *RedirectServer {
	s := &RedirectServer{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRedirectHTTPS())

	s.server = &http.Server{
		MaxHeaderBytes: 16 * 1024,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   5 * time.Second,
		IdleTimeout:    20 * time.Second,
		Handler:        mux,
	}

	s.Status = http.StatusFound
	s.IncludeRequest = true
	s.Logger = log.Printf
	return s
}

// ListenAndServe starts the redirect server
func (s *RedirectServer) ListenAndServe(addr string) error {
	s.server.Addr = addr
	return s.server.ListenAndServe()
}
