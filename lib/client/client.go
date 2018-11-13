package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/uber/makisu/lib/fileio"
	"github.com/uber/makisu/lib/log"
	"github.com/uber/makisu/lib/utils"
)

// MakisuClient is the struct that allows communication with a makisu worker.
type MakisuClient struct {
	LocalSharedPath  string
	WorkerSharedPath string

	WorkerLog func(line string)
	HTTPDo    func(req *http.Request) (*http.Response, error)
}

// New creates a new Makisu client that will talk to the worker available on the socket
// passed in.
func New(socket, localPath, workerPath string) *MakisuClient {
	cli := &http.Client{
		Transport: &http.Transport{
			Dial: func(p, a string) (net.Conn, error) {
				return net.Dial("unix", socket)
			},
		},
	}
	return &MakisuClient{
		LocalSharedPath:  localPath,
		WorkerSharedPath: workerPath,

		WorkerLog: func(line string) { fmt.Fprintf(os.Stderr, line+"\n") },
		HTTPDo:    cli.Do,
	}
}

// SetWorkerLog sets the function called on every worker log line.
func (cli *MakisuClient) SetWorkerLog(fn func(line string)) {
	cli.WorkerLog = fn
}

// Ready returns true if the worker is ready for work, and false if it is already performing
// a build.
func (cli *MakisuClient) Ready() (bool, error) {
	req, err := http.NewRequest("GET", "http://localhost/ready", nil)
	if err != nil {
		return false, err
	}
	resp, err := cli.HTTPDo(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}

// Build kicks off a build on the makisu worker at the context with the flags passed in.
func (cli *MakisuClient) Build(flags []string, context string) error {
	context, err := cli.prepareContext(context)
	if err != nil {
		return err
	}
	localContext := filepath.Join(cli.LocalSharedPath, context)
	workerContext := filepath.Join(cli.WorkerSharedPath, context)
	defer func() {
		log.Infof("Removing context after build: %s", localContext)
		os.RemoveAll(localContext)
	}()

	args := append(flags, workerContext)
	args = append([]string{"build"}, args...)

	content, _ := json.Marshal(args)
	reader := bytes.NewBuffer(content)
	req, err := http.NewRequest("POST", "http://localhost/build", reader)
	if err != nil {
		return err
	}

	resp, err := cli.HTTPDo(req)
	if err != nil {
		return err
	}

	return cli.readLines(resp.Body)
}

// Takes in the local path of the context, copies the files to a new directory inside the worker's
// mount namespace and returns the context path inside the shared mount location.
// Example: prepareContext("/home/joe/test/context") => context-12345
// This means that the context was copied over to <cmd.LocalSharedPath>/context-12345
func (cli *MakisuClient) prepareContext(context string) (string, error) {
	context, err := filepath.Abs(context)
	if err != nil {
		return "", err
	}

	rand := rand.New(rand.NewSource(time.Now().Unix()))
	targetContext := fmt.Sprintf("context-%d", rand.Intn(10000))
	targetPath := filepath.Join(cli.LocalSharedPath, targetContext)

	currUID, currGID, err := utils.GetUIDGID()
	if err != nil {
		return "", err
	}

	log.Infof("Copying context to worker filesystem: %s => %s", context, targetPath)
	start := time.Now()
	if err := fileio.NewCopier(nil).CopyDir(context, targetPath, currUID, currGID); err != nil {
		return "", err
	}
	log.Infof("Finished copying over context in %v", time.Since(start))
	return targetContext, nil
}

func (cli *MakisuClient) readLines(body io.ReadCloser) error {
	defer body.Close()
	reader := bufio.NewReader(body)
	for {
		line, _, err := reader.ReadLine()
		if err == io.EOF {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read build body: %v", err)
		}
		cli.WorkerLog(string(line))
	}
}
