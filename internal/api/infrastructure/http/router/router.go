package router

import "net/http"

type RouteGroup interface {
	RegisterRoutes(mux *http.ServeMux)
}

type Options struct {
	LearningEvents RouteGroup
}

func New(options Options) http.Handler {
	mux := http.NewServeMux()
	if options.LearningEvents != nil {
		options.LearningEvents.RegisterRoutes(mux)
	}
	return mux
}
