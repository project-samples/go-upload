package app

import (
	"context"
	"net/http"

	. "github.com/core-go/security"
	"github.com/gorilla/mux"
)

const (
	role     = "role"
	user     = "user"
	auditLog = "audit-log"
)

func Route(r *mux.Router, ctx context.Context, conf Root) error {
	app, err := NewApp(ctx, conf)
	if err != nil {
		return err
	}

	r.Use(app.AuthorizationHandler.HandleAuthorization)
	sec := &SecurityConfig{SecuritySkip: conf.SecuritySkip, Check: app.AuthorizationChecker.Check, Authorize: app.Authorizer.Authorize}

	Handle(r, "/health", app.HealthHandler.Check, GET)
	Handle(r, "/authenticate", app.AuthenticationHandler.Authenticate, POST)

	r.Handle("/code/{code}", app.AuthorizationChecker.Check(http.HandlerFunc(app.CodeHandler.Load))).Methods(GET)

	uploadHandler := app.UploadHandler
	uploads := r.PathPrefix("/uploads").Subrouter()
	image := r.PathPrefix("/image").Subrouter()
	HandleWithSecurity(sec, uploads, "", uploadHandler.All, "upload", ActionRead, GET)
	HandleWithSecurity(sec, uploads, "/{userId}", uploadHandler.Load, "upload", ActionRead, GET)
	HandleWithSecurity(sec, uploads, "/youtube", uploadHandler.Create, "upload", ActionWrite, POST)
	HandleWithSecurity(sec, uploads, "", uploadHandler.Update, "upload", ActionWrite, PATCH)
	HandleWithSecurity(sec, uploads, "/youtube", uploadHandler.Delete, "upload", ActionWrite, DELETE)
	HandleWithSecurity(sec, image, "/users/{userId}", uploadHandler.LoadImage, "user", ActionRead, GET)
	HandleWithSecurity(sec, uploads, "", uploadHandler.UploadFile, "upload", ActionWrite, POST)
	HandleWithSecurity(sec, uploads, "", uploadHandler.DeleteFile, "upload", ActionWrite, DELETE)

	return nil
}

func Handle(r *mux.Router, path string, f func(http.ResponseWriter, *http.Request), methods ...string) *mux.Route {
	return r.HandleFunc(path, f).Methods(methods...)
}
func HandleWithSecurity(authorizer *SecurityConfig, r *mux.Router, path string, f func(http.ResponseWriter, *http.Request), menuId string, action int32, methods ...string) *mux.Route {
	finalHandler := http.HandlerFunc(f)
	if authorizer.SecuritySkip {
		return r.HandleFunc(path, finalHandler).Methods(methods...)
	}
	authorize := func(next http.Handler) http.Handler {
		return authorizer.Authorize(next, menuId, action)
	}
	return r.Handle(path, authorizer.Check(authorize(finalHandler))).Methods(methods...)
}
