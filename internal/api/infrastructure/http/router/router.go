package router

import "net/http"

type RouteGroup interface {
	RegisterRoutes(mux *http.ServeMux)
}

type Options struct {
	LearningEvents RouteGroup
	WatchProgress  RouteGroup
}

func New(options Options) http.Handler {
	mux := http.NewServeMux()
	if options.LearningEvents != nil {
		options.LearningEvents.RegisterRoutes(mux)
	}
	if options.WatchProgress != nil {
		options.WatchProgress.RegisterRoutes(mux)
	}
	return mux
}
