package render

import (
	"bytes"
	"fmt"

	"github.com/indiereign/shift72-kibble/kibble/models"
	"github.com/indiereign/shift72-kibble/kibble/utils"

	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	fsnotify "gopkg.in/fsnotify.v1"
)

var embed = `
<script>
(function(){
	var etag = '';
  function checkForChanges() {
    xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
			if (this.readyState == this.HEADERS_RECEIVED) {
        etag = this.getResponseHeader("Etag");
      }
      if (this.readyState == 4 && this.status == 200) {
        location.reload(true);
      }
    };
    xhttp.open("GET", "/kibble/live_reload", true);
		xhttp.setRequestHeader("If-Modified-Since", etag);
    xhttp.send();
    setTimeout(checkForChanges, 3000);
  }
  setTimeout(checkForChanges, 3000);
})();
</script>
`

var ignorePaths = []string{
	".git",
	".kibble",
	"node_modules",
	"npm-debug.log",
	"package.json",
}

// LiveReload -
type LiveReload struct {
	lastModified time.Time
	logReader    utils.LogReader
	sourcePath   string
	config       models.LiveReloadConfig
}

// WrapperResponseWriter - wraps request
// intercepts all write calls so as to append the live reload script
type WrapperResponseWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	buf         bytes.Buffer
	prefixBuf   bytes.Buffer
}

// NewWrapperResponseWriter - create a new response writer
func NewWrapperResponseWriter(w http.ResponseWriter) *WrapperResponseWriter {
	return &WrapperResponseWriter{ResponseWriter: w}
}

// Status - get the status
func (w *WrapperResponseWriter) Status() int {
	return w.status
}

// Write - wrap the write
func (w *WrapperResponseWriter) Write(p []byte) (n int, err error) {
	w.buf.Write(p)
	return len(p), nil
}

// PrefixWithLogs - write the logs to the head of the page
func (w *WrapperResponseWriter) PrefixWithLogs(logs []string) {
	w.prefixBuf.Write([]byte("<div>"))
	for _, s := range logs {
		w.prefixBuf.Write([]byte(fmt.Sprintf("<pre>%s</pre>", s)))
	}
	w.prefixBuf.Write([]byte("</div>"))
}

// Done - called when are ready to return a result
func (w *WrapperResponseWriter) Done() (n int, err error) {
	w.Header().Set("Content-Length", strconv.Itoa(w.buf.Len()+w.prefixBuf.Len()))
	w.ResponseWriter.WriteHeader(w.status)
	w.ResponseWriter.Write(w.prefixBuf.Bytes())
	return w.ResponseWriter.Write(w.buf.Bytes())
}

// WriteHeader - wrap the write header
func (w *WrapperResponseWriter) WriteHeader(code int) {
	w.status = code
}

// GetMiddleware - return a handler
func (live *LiveReload) GetMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ww := NewWrapperResponseWriter(w)
		next.ServeHTTP(ww, r)

		if strings.HasSuffix(r.RequestURI, "/") ||
			strings.HasSuffix(r.RequestURI, "/index.html") {

			if ww.Status() == 200 {
				ww.PrefixWithLogs(live.logReader.Logs())
				ww.Write([]byte(embed))

			}
		}

		ww.Done()
	})
}

// Handler - handle the live reload
func (live *LiveReload) Handler(w http.ResponseWriter, req *http.Request) {
	matchEtag := req.Header.Get("If-Modified-Since")

	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Etag", live.lastModified.String())

	if matchEtag == live.lastModified.String() || matchEtag == "" {
		w.WriteHeader(http.StatusNotModified)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	w.Write([]byte(fmt.Sprintf("last modified: %s", live.lastModified.String())))
}

// StartLiveReload - start the process to watch the files and wait for a reload
func (live *LiveReload) StartLiveReload(port int32, fn func()) {

	rendered := make(chan bool)

	// wait for changes
	changesChannel := make(chan bool)
	go func() {
		log.Info("Starting live reload")

		for _ = range changesChannel {
			now := time.Now()
			// throttle the amount of changes, due to some editors
			// *cough* Sublime Text *cough* sending multiple WRITES for 1 file
			if !live.lastModified.IsZero() && now.Sub(live.lastModified).Seconds() <= 1 {
				log.Debug("Ignoring multiple changes")
				continue
			}

			fn()
			live.lastModified = time.Now()

			// non blocking send
			select {
			case rendered <- true:
			default:
			}
		}
	}()

	// launch the browser
	go func() {

		// wait for the channel to be rendered
		<-rendered

		cmd := exec.Command("open", fmt.Sprintf("http://localhost:%d/", port))
		err := cmd.Start()
		if err != nil {
			log.Error("Watcher: ", err)
		}
	}()

	// useful to trigger one new reload
	changesChannel <- true

	live.selectFilesToWatch(changesChannel)
}

func (live *LiveReload) selectFilesToWatch(changesChannel chan bool) {

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	// listen for fs events and pass via channel
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Critical("change (%s) detected: %s", event.Op, event.Name)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Debugf("change (%s) detected: %s", event.Op, event.Name)
					changesChannel <- true
				}
			case err = <-watcher.Errors:
				log.Error("Watcher: ", err)
			}
		}
	}()

	// make a single list of things to ignore,
	// so its not generated everytime we do a file compare
	patterns := append(ignorePaths, live.config.IgnoredPaths...)
	for i, p := range patterns {
		patterns[i] = filepath.Join(live.sourcePath, p)
	}

	log.Debug("Ignoring - ", patterns)

	// search the path for files that might have changed
	err = filepath.Walk(live.sourcePath, func(path string, f os.FileInfo, err error) error {
		if live.ignorePath(path, patterns) {
			return nil
		}

		if f.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				log.Error("Watcher: ", err)
			}
		}
		return nil
	})

	if err != nil {
		log.Error("Watcher: ", err)
	}
}

func (live LiveReload) ignorePath(name string, patterns []string) bool {
	// check default ignored file patterns
	for _, c := range patterns {
		if filePathMatches(name, c) {
			return true
		}
	}

	return false
}

func filePathMatches(path string, pattern string) bool {
	isMatch, err := filepath.Match(pattern, path)
	if err != nil {
		log.Errorf("Watcher failed matching %s to %s. %s", path, pattern, err.Error())
	}
	// support both file globs and simple dir names (which the `filepath.Match` command seems to not support).
	if isMatch || strings.HasPrefix(path, pattern) {
		return true
	}

	return false
}
