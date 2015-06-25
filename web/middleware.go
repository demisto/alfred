package web

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/server/util"
	"github.com/gorilla/context"
)

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.WithField("error", err).Warn("Recovered from error")
				WriteError(w, ErrInternalServer)
			}
		}()

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (l *loggingResponseWriter) WriteHeader(status int) {
	l.status = status
	l.ResponseWriter.WriteHeader(status)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		lw := &loggingResponseWriter{w, 200}
		t1 := time.Now()
		next.ServeHTTP(lw, r)
		t2 := time.Now()
		log.Infof("[%s] %q %v %v\n", r.Method, r.URL.String(), lw.status, t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func acceptHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept"), "application/json") {
			log.Warn("Request without accept header received")
			WriteError(w, ErrNotAcceptable)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func contentTypeHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			log.Warn("Request without proper content type received")
			WriteError(w, ErrUnsupportedMediaType)
			return
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func bodyHandler(v interface{}) func(http.Handler) http.Handler {
	t := reflect.TypeOf(v)

	m := func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			val := reflect.New(t).Interface()
			err := json.NewDecoder(r.Body).Decode(val)

			if err != nil {
				log.WithFields(log.Fields{"body": r.Body, "err": err}).Warn("Error handling body")
				WriteError(w, ErrBadRequest)
				return
			}

			if next != nil {
				context.Set(r, "body", val)
				next.ServeHTTP(w, r)
			}
		}

		return http.HandlerFunc(fn)
	}

	return m
}

const (
	// xsrfCookie is the name of the XSRF cookie
	xsrfCookie = `XSRF`
	// xsrfHeader is the name of the expected header
	xsrfHeader = `X-XSRF-TOKEN`
	// noXsrfAllowed is the error message
	noXSRFAllowed = `No XSRF Allowed`
)

// Handle CSRF protection
func csrfHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		csrf, err := r.Cookie(xsrfCookie)
		csrfHeader := r.Header.Get(xsrfHeader)
		ok := false
		secure := conf.Options.Env == "PROD" || conf.Options.Env == "TEST"
		pass := conf.Options.Security.SessionKey
		// If it is an idempotent method, set the cookie
		if r.Method == "GET" || r.Method == "HEAD" {
			if err != nil {
				val, cErr := util.Encrypt(noXSRFAllowed+time.Now().String(), pass)
				if cErr == nil {
					http.SetCookie(w, &http.Cookie{Name: xsrfCookie, Value: val, Path: "/", Expires: time.Now().Add(365 * 24 * time.Hour), MaxAge: 365 * 24 * 60 * 60, Secure: secure, HttpOnly: false})
				} else {
					log.WithField("error", cErr).Error("Unable to generate CSRF")
				}
			}
			ok = true
		} else if err == nil && csrf.Value == csrfHeader {
			val, cErr := util.Decrypt(csrfHeader, pass)
			if cErr == nil && strings.HasPrefix(val, noXSRFAllowed) {
				ok = true
			}
		}
		if ok {
			next.ServeHTTP(w, r)
		} else {
			WriteError(w, ErrCSRF)
		}
	}
	return http.HandlerFunc(fn)
}

const (
	sessionCookie = `SES`
)

func (ac *AppContext) authHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookie)
		// No session, bye bye
		if err != nil {
			log.Info("Access to authenticated service without session")
			WriteError(w, ErrAuth)
			return
		}
		var sess session
		err = util.DecryptJSON(cookie.Value, conf.Options.Security.SessionKey, &sess)
		if err != nil {
			log.WithFields(log.Fields{"cookie": cookie.Value, "error": err}).Warn("Unable to decrypt encrypted session")
			WriteError(w, ErrAuth)
			return
		}
		// If the session is no longer valid
		if time.Since(sess.When) > time.Duration(conf.Options.Security.Timeout)*time.Minute {
			log.Debug("Session timeout")
			WriteError(w, ErrAuth)
			return
		}
		context.Set(r, "session", &sess)
		log.Debugf("User %v in request", sess.User)
		u, err := ac.r.User(sess.UserID)
		if err != nil {
			log.WithFields(log.Fields{"username": sess.User, "id": sess.UserID, "error": err}).Warn("Unable to load user from repository")
			panic(err)
		}
		context.Set(r, "user", u)
		// Set the new cookie for the user with the new timeout
		sess.When = time.Now()
		secure := conf.Options.Env == "PROD" || conf.Options.Env == "TEST"
		val, _ := util.EncryptJSON(&sess, conf.Options.Security.SessionKey)
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookie,
			Value:    val,
			Path:     "/",
			Expires:  time.Now().Add(time.Duration(conf.Options.Security.Timeout) * time.Minute),
			MaxAge:   conf.Options.Security.Timeout * 60,
			Secure:   secure,
			HttpOnly: true})
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
