package main

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/qor5/admin/v3/docs/docsrc/examples/examples_presets"
	"github.com/qor5/web/v3"
	"github.com/qor5/web/v3/examples"
	"github.com/theplant/osenv"
)

var port = osenv.Get("PORT", "The port to serve on", "7800")

func main() {
	fmt.Println("Starting docs at :" + port)
	mux := http.NewServeMux()
	im := &examples.IndexMux{Mux: http.NewServeMux()}
	examples_presets.SamplesHandler(im, "/samples")
	mux.Handle("/samples/",
		middleware.Logger(
			middleware.RequestID(
				im.Mux,
			),
		),
	)
	mux.Handle("/", web.New().Page(im.Page))

	err := http.ListenAndServe(":"+port, mux)
	if err != nil {
		panic(err)
	}
}
