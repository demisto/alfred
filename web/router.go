package web

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/demisto/alfred/conf"
	"github.com/demisto/alfred/domain"
	"github.com/gorilla/context"
	"github.com/julienschmidt/httprouter"
	"github.com/justinas/alice"
)

// Main handlers
var public string

func pageHandler(file string) func(w http.ResponseWriter, r *http.Request) {
	m := func(w http.ResponseWriter, r *http.Request) {
		log.Debugf("Looking for file %s\n", file)
		f, err := FS(conf.IsDev()).Open(file)
		if err != nil {
			log.Warnf("Could not find file %s - %v", file, err)
			WriteError(w, ErrNotFound)
			return
		}
		stat, err := f.Stat()
		if err != nil {
			log.Warn("Could not stat file %s - %v", file, err)
			WriteError(w, ErrNotFound)
			return
		}
		http.ServeContent(w, r, file, stat.ModTime(), f)
	}
	return m
}

// Router

// Router handles the web requests routing
type Router struct {
	*httprouter.Router
}

// Get handles GET requests
func (r *Router) Get(path string, handler http.Handler) {
	r.GET(path, wrapHandler(handler))
}

// Post handles POST requests
func (r *Router) Post(path string, handler http.Handler) {
	r.POST(path, wrapHandler(handler))
}

// Put handles PUt requests
func (r *Router) Put(path string, handler http.Handler) {
	r.PUT(path, wrapHandler(handler))
}

// Delete handles DELETE requests
func (r *Router) Delete(path string, handler http.Handler) {
	r.DELETE(path, wrapHandler(handler))
}

// New creates a new router
func New(appC *AppContext) *Router {
	r := &Router{httprouter.New()}
	staticHandlers := alice.New(context.ClearHandler, loggingHandler, csrfHandler, recoverHandler)
	commonHandlers := staticHandlers.Append(acceptHandler)
	authHandlers := commonHandlers.Append(appC.authHandler)
	// Security
	r.Get("/oauth", staticHandlers.ThenFunc(appC.initiateOAuth))
	r.Get("/auth", staticHandlers.ThenFunc(appC.loginOAuth))
	r.Get("/logout", staticHandlers.ThenFunc(appC.logout))
	r.Get("/user", authHandlers.ThenFunc(appC.currUser))
	r.Get("/info", authHandlers.ThenFunc(appC.info))
	r.Post("/match", authHandlers.Append(contentTypeHandler, bodyHandler(regexpMatch{})).ThenFunc(appC.match))
	r.Post("/save", authHandlers.Append(contentTypeHandler, bodyHandler(domain.Configuration{})).ThenFunc(appC.save))
	r.Get("/work", commonHandlers.ThenFunc(appC.work))
	// Static
	r.Get("/", staticHandlers.ThenFunc(pageHandler("/index.html")))
	r.Get("/conf", staticHandlers.ThenFunc(pageHandler("/conf.html")))
	r.Get("/details", staticHandlers.ThenFunc(pageHandler("/details.html")))
	r.ServeFiles("/css/*filepath", Dir(conf.IsDev(), "/css/"))
	r.ServeFiles("/img/*filepath", Dir(conf.IsDev(), "/img/"))
	r.ServeFiles("/js/*filepath", Dir(conf.IsDev(), "/js/"))
	r.ServeFiles("/vendor/*filepath", Dir(conf.IsDev(), "/vendor/"))
	r.ServeFiles("/video/*filepath", Dir(conf.IsDev(), "/video/"))
	return r
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}

func redirectToHTTPS(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, conf.Options.ExternalAddress+r.RequestURI, http.StatusMovedPermanently)
}

// Serve the routes based on configuration
func (r *Router) Serve() {
	var err error
	if conf.Options.SSL.Cert != "" {
		// First, listen on the HTTP address with redirect
		go func() {
			err := http.ListenAndServe(conf.Options.HTTPAddress, http.HandlerFunc(redirectToHTTPS))
			if err != nil {
				log.Fatal(err)
			}
		}()
		addr := conf.Options.Address
		if addr == "" {
			addr = ":https"
		}
		server := &http.Server{Addr: conf.Options.Address, Handler: r}
		config := &tls.Config{NextProtos: []string{"http/1.1"}}
		config.Certificates = make([]tls.Certificate, 1)
		config.Certificates[0], err = tls.X509KeyPair([]byte(conf.Options.SSL.Cert), []byte(conf.Options.SSL.Key))
		if err != nil {
			log.Fatal(err)
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			log.Fatal(err)
		}
		tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
		err = server.Serve(tlsListener)
	} else {
		err = http.ListenAndServe(conf.Options.Address, r)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		context.Set(r, "params", ps)
		h.ServeHTTP(w, r)
	}
}
