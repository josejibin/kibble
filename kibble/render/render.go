package render

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/CloudyKit/jet"
	"github.com/indiereign/shift72-kibble/kibble/api"
	"github.com/indiereign/shift72-kibble/kibble/config"
	"github.com/indiereign/shift72-kibble/kibble/datastore"
	"github.com/indiereign/shift72-kibble/kibble/models"
	"github.com/nicksnyder/go-i18n/i18n"
)

var rootPath = "./.kibble/build"
var publicFolder = "public"

// Watch -
func Watch(runAsAdmin bool, verbose bool, port int32) {

	liveReload := LiveReload{}

	mux := http.NewServeMux()
	mux.HandleFunc("/kibble/live_reload", liveReload.Handler)
	mux.Handle("/", liveReload.GetMiddleware(http.FileServer(http.Dir(rootPath))))

	liveReload.StartLiveReload(func() {
		Render(runAsAdmin, verbose)
	})

	// launch the browser
	go func() {
		time.Sleep(500 * time.Millisecond)

		waitForIndexFile()

		cmd := exec.Command("open", fmt.Sprintf("http://localhost:%d/", port))
		err := cmd.Start()
		if err != nil {
			fmt.Println(err)
		}
	}()

	fmt.Printf("listening on %d\n", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
	if err != nil {
		fmt.Println(err)
	}
}

func waitForIndexFile() {
	path := path.Join(rootPath, "index.html")

	for i := 0; i < 15; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := os.Stat(path)
		if !os.IsNotExist(err) {
			break
		}
	}
}

// Render - render the files
func Render(runAsAdmin bool, verbose bool) {

	datastore.Init()

	cfg := config.LoadConfig(runAsAdmin)

	api.CheckAdminCredentials(cfg, runAsAdmin)

	site, err := api.LoadSite(cfg)
	if err != nil {
		fmt.Printf("Site load failed: %s", err)
		return
	}

	routeRegistry := models.NewRouteRegistryFromConfig(cfg)

	renderer := FileRenderer{
		rootPath:    rootPath,
		showSummary: verbose,
	}
	renderer.Initialise()

	start := time.Now()

	err = Sass(
		path.Join("styles", "main.scss"),
		path.Join(rootPath, "styles", "main.css"))
	if err != nil {
		fmt.Printf("Sass rendering failed: %s", err)
		return
	}

	fmt.Printf("Sass render time: %s\n", time.Now().Sub(start))

	for lang, locale := range cfg.Languages {

		T, err := i18n.Tfunc(locale, cfg.DefaultLanguage)
		if err != nil {
			fmt.Println(err)
		}

		ctx := models.RenderContext{
			RoutePrefix: "",
			Site:        site,
			Language:    lang,
		}

		if lang != cfg.DefaultLanguage {
			ctx.RoutePrefix = fmt.Sprintf("/%s", lang)
		}

		// set the template view
		renderer.view = models.CreateTemplateView(routeRegistry, T, ctx, "./")

		// render static files
		files, _ := filepath.Glob("*.jet")
		for _, f := range files {
			filePath := fmt.Sprintf("%s/%s", ctx.RoutePrefix, strings.Replace(f, ".jet", "", 1))

			route := &models.Route{
				TemplatePath: f,
			}

			data := jet.VarMap{}
			data.Set("site", site)
			renderer.Render(route, filePath, data)
		}

		for _, route := range routeRegistry.GetAll() {

			ctx.Route = route
			if route.ResolvedDataSouce != nil {
				route.ResolvedDataSouce.Iterator(ctx, renderer)
			}
		}
	}

	stop := time.Now()

	fmt.Printf("\nTotal render time: %s\n", stop.Sub(start))
}
