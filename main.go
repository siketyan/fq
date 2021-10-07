package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/djherbis/times"
)

type File struct {
	Path           string      `json:"path"`
	Name           string      `json:"name"`
	Mode           fs.FileMode `json:"mode"`
	CreatedAt      *time.Time  `json:"created_at"`
	ChangedAt      *time.Time  `json:"changed_at"`
	ModifiedAt     time.Time   `json:"modified_at"`
	LastAccessedAt time.Time   `json:"last_accessed_at"`
}

type Jq struct {
	Cmd *exec.Cmd
}

func startJq(args ...string) (*Jq, io.WriteCloser, error) {
	cmd := exec.Command("jq", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	writer, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, newError("I/O pipe error", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, newError("Failed to run jq command", err)
	}

	return &Jq{Cmd: cmd}, writer, nil
}

func (j *Jq) wait(errChan chan error) {
	if err := j.Cmd.Wait(); err != nil {
		errChan <- newError("Failed to wait jq to exit", err)
	} else {
		errChan <- nil
	}
}

type Error struct {
	Message string
	Cause   error
}

func newError(msg string, cause error) error {
	return &Error{
		Message: msg,
		Cause:   cause,
	}
}

func (e *Error) Error() string {
	if e.Cause == nil {
		return e.Message
	}

	return fmt.Sprintf("%s (Caused by: %s)", e.Message, e.Cause.Error())
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		die(newError("Any argument must be passed", nil))

		return
	}

	if err := run(args[0], args[1:]...); err != nil {
		die(err)

		return
	}
}

func die(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "Error: %s; aborting.", err.Error())
	os.Exit(1)
}

func run(glob string, jqArgs ...string) error {
	files, err := fq(glob)
	if err != nil {
		return newError("Failed to query files from the glob", err)
	}

	var writer io.WriteCloser
	var errChan chan error
	if 0 < len(jqArgs) {
		jq, w, err := startJq(jqArgs...)
		if err != nil {
			return newError("Failed to start jq", err)
		}

		writer = w
		errChan = make(chan error)

		go jq.wait(errChan)
	} else {
		writer = os.Stdout
	}

	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(files); err != nil {
		return newError("Failed to encode to JSON", err)
	}

	if err := writer.Close(); err != nil {
		return newError("I/O error", err)
	}

	if errChan != nil {
		if err := <-errChan; err != nil {
			return err
		}
	}

	return nil
}

func fq(glob string) ([]*File, error) {
	files := make([]*File, 0)
	paths, err := filepath.Glob(glob)
	if err != nil {
		return nil, newError("Failed to match from the glob", err)
	}

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			return nil, newError("Failed to stat the file", err)
		}

		if info.IsDir() {
			continue
		}

		t, err := times.Stat(path)
		if err != nil {
			return nil, newError("Failed to fetch timestamps of the file", err)
		}

		file := &File{
			Path:           path,
			Name:           info.Name(),
			Mode:           info.Mode(),
			ModifiedAt:     info.ModTime(),
			LastAccessedAt: t.AccessTime(),
		}

		if t.HasBirthTime() {
			birthTime := t.BirthTime()
			file.CreatedAt = &birthTime
		}

		if t.HasChangeTime() {
			changeTime := t.ChangeTime()
			file.ChangedAt = &changeTime
		}

		files = append(files, file)
	}

	return files, nil
}
