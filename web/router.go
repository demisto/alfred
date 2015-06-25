package web

import (
	"fmt"
	"net/http"
	"os"

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
		http.ServeFile(w, r, public+file)
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

func handlePublicPath(pubPath string) {
	switch {
	// absolute path
	case len(pubPath) > 1 && (pubPath[0] == '/' || pubPath[0] == '\\'):
		public = pubPath
	// absolute path win
	case len(pubPath) > 2 && pubPath[1] == ':':
		public = pubPath
	// relative
	case len(pubPath) > 1 && pubPath[0] == '.':
		public = pubPath
	default:
		public = "./" + pubPath
	}
	if pubPath[len(pubPath)-1] == '/' || pubPath[len(pubPath)-1] == '\\' {
		public = pubPath
	} else {
		public = fmt.Sprintf("%s%c", pubPath, os.PathSeparator)
	}
	log.Infof("Using public path %v", public)
	conf.PublicPath = public
}

// New creates a new router
func New(appC *AppContext, pubPath string) *Router {
	handlePublicPath(pubPath)
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
	r.Post("/save", authHandlers.Append(contentTypeHandler, bodyHandler(domain.Configuration{})).ThenFunc(appC.save))
	// Static
	r.Get("/", staticHandlers.ThenFunc(pageHandler("index.html")))
	r.Get("/conf", staticHandlers.ThenFunc(pageHandler("conf.html")))
	r.ServeFiles("/css/*filepath", http.Dir(public+"css"))
	r.ServeFiles("/img/*filepath", http.Dir(public+"img"))
	r.ServeFiles("/js/*filepath", http.Dir(public+"js"))
	r.ServeFiles("/vendor/*filepath", http.Dir(public+"vendor"))
	r.ServeFiles("/video/*filepath", http.Dir(public+"video"))
	return r
}

func wrapHandler(h http.Handler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		context.Set(r, "params", ps)
		h.ServeHTTP(w, r)
	}
}
