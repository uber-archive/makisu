package cli

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/apourchet/commander"
	"github.com/uber/makisu/lib/log"
	"go.uber.org/atomic"
)

// WorkerApplication contains the bindings for the `makisu-wokrer listen` command.
type WorkerApplication struct {
	ApplicationFlags `commander:"flagstruct"`
	ListenFlags      `commander:"flagstruct=listen"`
}

// ListenFlags contains all of the flags for `makisu listen ...`
type ListenFlags struct {
	SocketPath string `commander:"flag=s,The absolute path of the unix socket that makisu will listen on"`
	building   *atomic.Bool
}

// NewWorkerApplication returns a new worker application for the listen command.
func NewWorkerApplication() *WorkerApplication {
	return &WorkerApplication{
		ApplicationFlags: defaultApplicationFlags(),
		ListenFlags:      newListenFlags(),
	}
}

func newListenFlags() ListenFlags {
	return ListenFlags{
		SocketPath: "/makisu-socket/makisu.sock",
		building:   atomic.NewBool(false),
	}
}

// BuildRequest is the expected structure of the JSON body of http requests coming in on the socket.
// Example body of a BuildRequest:
//    ["build", "-t", "myimage:latest", "/context"]
type BuildRequest []string

// Listen creates the directory structures and the makisu socket, then it
// starts accepting http requests on that socket.
func (cmd ListenFlags) Listen() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", cmd.ready)
	mux.HandleFunc("/exit", cmd.exit)
	mux.HandleFunc("/build", cmd.build)

	if err := os.MkdirAll(path.Dir(cmd.SocketPath), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory to socket %s: %v", cmd.SocketPath, err)
	}

	lis, err := net.Listen("unix", cmd.SocketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket %s: %v", cmd.SocketPath, err)
	}
	log.Infof("Listening for build requests on unix socket %s", cmd.SocketPath)

	server := http.Server{Handler: mux}
	if err := server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve on unix socket: %v", err)
	}
	return nil
}

func (cmd ListenFlags) ready(rw http.ResponseWriter, req *http.Request) {
	if cmd.building.Load() {
		rw.WriteHeader(http.StatusConflict)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (cmd ListenFlags) exit(rw http.ResponseWriter, req *http.Request) {
	if ok := cmd.building.CAS(false, true); !ok {
		rw.WriteHeader(http.StatusConflict)
		rw.Write([]byte("Already processing a request"))
		return
	}
	rw.WriteHeader(http.StatusOK)
	go func() {
		time.Sleep(5 * time.Second)
		os.Exit(0)
	}()
}

func (cmd ListenFlags) build(rw http.ResponseWriter, req *http.Request) {
	if ok := cmd.building.CAS(false, true); !ok {
		rw.WriteHeader(http.StatusConflict)
		rw.Write([]byte("Already processing a request"))
		return
	}
	defer cmd.building.Store(false)

	log.Infof("Serving build request")
	args := &BuildRequest{}
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, "%s\n", err.Error())
		return
	} else if err := json.Unmarshal(body, args); err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(rw, "%s\n", err.Error())
		return
	}
	log.Infof("Build arguments passed in: %s", string(body))

	r, newStderr, err := os.Pipe()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(rw, "%s\n", err.Error())
		return
	}

	log.Infof("Piping stderr to response")
	oldLogger := log.GetLogger()
	os.Stderr = newStderr
	done := make(chan bool, 0)

	defer func() {
		newStderr.Close()
		<-done
		log.SetLogger(oldLogger)
		log.Infof("Build request served")
	}()

	go func() {
		defer func() { done <- true }()
		var fl http.Flusher
		if f, ok := rw.(http.Flusher); ok {
			fl = f
		}
		flushLines(r, rw, fl)
	}()

	rw.WriteHeader(http.StatusOK)
	log.Infof("Starting build")

	commander := commander.New()
	commander.FlagErrorHandling = flag.ContinueOnError
	app := NewBuildApplication()
	app.AllowModifyFS = true
	if err := commander.RunCLI(app, *args); err != nil {
		log.With("build_code", "1").Errorf("Failed to run CLI: %v", err)
		return
	} else if err := app.Cleanup(); err != nil {
		log.With("build_code", "1").Errorf("Failed to cleanup: %v", err)
		return
	}
	log.With("build_code", "0").Infof("Build exited successfully")
}

func flushLines(r io.Reader, w io.Writer, fl http.Flusher) {
	reader := bufio.NewReader(r)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			return
		} else if err != nil {
			return
		}
		line = append(line, '\n')
		w.Write(line)
		if fl != nil {
			fl.Flush()
		}
	}
}
