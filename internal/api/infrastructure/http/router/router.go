package router

import "net/http"

type RouteGroup interface {
	RegisterRoutes(mux *http.ServeMux)
}

type Options struct {
	Feed              RouteGroup
	EndQuiz           RouteGroup
	VideoInteractions RouteGroup
	LearningEvents    RouteGroup
	WatchProgress     RouteGroup
}

func New(options Options) http.Handler {
	mux := http.NewServeMux()
	if options.Feed != nil {
		options.Feed.RegisterRoutes(mux)
	}
	if options.EndQuiz != nil {
		options.EndQuiz.RegisterRoutes(mux)
	}
	if options.VideoInteractions != nil {
		options.VideoInteractions.RegisterRoutes(mux)
	}
	if options.LearningEvents != nil {
		options.LearningEvents.RegisterRoutes(mux)
	}
	if options.WatchProgress != nil {
		options.WatchProgress.RegisterRoutes(mux)
	}
	return mux
}
