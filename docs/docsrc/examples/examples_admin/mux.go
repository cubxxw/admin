package examples_admin

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/qor5/admin/v3/docs/docsrc/examples"
	"github.com/qor5/admin/v3/docs/docsrc/examples/examples_presets"
	"github.com/qor5/admin/v3/docs/docsrc/examples/examples_vuetify"
	webexamples "github.com/qor5/web/v3/examples"
)

func Mux(mux *http.ServeMux) http.Handler {
	examples_vuetify.Mux(mux)

	im := &webexamples.IndexMux{Mux: http.NewServeMux()}
	SamplesHandler(im)

	mux.Handle("/examples/",
		middleware.Logger(
			middleware.RequestID(
				im.Mux,
			),
		),
	)

	return mux
}

func SamplesHandler(mux webexamples.Muxer) {
	examples_vuetify.SamplesHandler(mux)
	examples_presets.SamplesHandler(mux)

	examples.AddPresetExample(mux, ListingExample)
	examples.AddPresetExample(mux, WorkerExample)
	examples.AddPresetExample(mux, ActionWorkerExample)
	examples.AddPresetExample(mux, InternationalizationExample)
	examples.AddPresetExample(mux, LocalizationExample)
	examples.AddPresetExample(mux, PublishExample)
	examples.AddPresetExample(mux, SEOExampleBasic)
	examples.AddPresetExample(mux, ActivityExample)
	examples.AddPresetExample(mux, PageBuilderExample)
	examples.AddPresetExample(mux, MediaExample)
}
