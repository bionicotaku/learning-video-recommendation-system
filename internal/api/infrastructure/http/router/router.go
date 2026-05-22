package router

import "net/http"

type RouteGroup interface {
	RegisterRoutes(mux *http.ServeMux)
}

type Options struct {
	Feed              RouteGroup
	EndQuiz           RouteGroup
	UnitCollections   RouteGroup
	VideoInteractions RouteGroup
	LearningEvents    RouteGroup
	WatchProgress     RouteGroup
	UnitProgress      RouteGroup
	Me                RouteGroup
	Feedback          RouteGroup
	LegalDocuments    RouteGroup
}

func New(options Options) http.Handler {
	mux := http.NewServeMux()
	if options.Feed != nil {
		options.Feed.RegisterRoutes(mux)
	}
	if options.EndQuiz != nil {
		options.EndQuiz.RegisterRoutes(mux)
	}
	if options.UnitCollections != nil {
		options.UnitCollections.RegisterRoutes(mux)
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
	if options.UnitProgress != nil {
		options.UnitProgress.RegisterRoutes(mux)
	}
	if options.Me != nil {
		options.Me.RegisterRoutes(mux)
	}
	if options.Feedback != nil {
		options.Feedback.RegisterRoutes(mux)
	}
	if options.LegalDocuments != nil {
		options.LegalDocuments.RegisterRoutes(mux)
	}
	return mux
}
