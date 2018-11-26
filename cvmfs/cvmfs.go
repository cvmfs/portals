package cvmfs

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cvmfs/portals/log"

	copy "github.com/otiai10/copy"
	logrus "github.com/sirupsen/logrus"
)

type FSModification interface {
	Repository() string
	Apply() error
	CleanUp()
}

type Add struct {
	repository  string
	source      string
	destination string
}

func (add Add) Repository() string {
	return add.repository
}

func (add Add) Apply() error {
	l := log.Decorate(map[string]string{
		"Action":      "Apply",
		"repository":  add.Repository(),
		"source":      add.source,
		"destination": add.getCompleteDestinationPath(),
	})
	sourceStat, err := os.Lstat(add.source)
	if err != nil {
		l(log.LogE(err)).Error("Error in stating the file")
		return err
	}
	mode := sourceStat.Mode()
	if mode.IsDir() {
		err = add.addDirectory()
		if err != nil {
			l(log.LogE(err)).Error("Error in adding the directory")
		}
		return err
	}
	if mode.IsRegular() {
		err = add.addRegular()
		if err != nil {
			l(log.LogE(err)).Error("Error in adding the regular file")
		}
		return err
	}
	if mode&os.ModeSymlink != 0 {
		err = add.addSymlink()
		if err != nil {
			l(log.LogE(err)).Error("Error in adding the symlink")
		}
		return err
	}
	err = fmt.Errorf("Apply FSModification but source is neither a directory, nor a regular file nor a symlink")
	l(log.LogE(err)).Error("Type of file not supported")
	return err
}

func (add Add) CleanUp() {
	os.RemoveAll(add.source)
}

func (add Add) getCompleteDestinationPath() string {
	return filepath.Join("/", "cvmfs", add.Repository(), add.destination)
}

func (add Add) addDirectory() error {
	l := log.Decorate(map[string]string{
		"Action":      "Add Directory",
		"repository":  add.Repository(),
		"source":      add.source,
		"destination": add.getCompleteDestinationPath(),
	})
	path := add.getCompleteDestinationPath()
	os.RemoveAll(path)
	err := os.MkdirAll(path, 0666)
	if err != nil {
		l(log.LogE(err)).Warning("Error in creating the directory where to copy the singularity")
	}
	err = copy.Copy(add.source, path)
	l(log.LogE(err)).Error("Error in copying the source directory into the destination one")

	return err
}
func (add Add) addRegular() error { return nil }
func (add Add) addSymlink() error { return nil }

type execCmd struct {
	cmd *exec.Cmd
	err io.ReadCloser
	out io.ReadCloser
}

func ExecCommand(input ...string) *execCmd {
	l := log.Decorate(map[string]string{
		"Action":  "creating command",
		"Command": strings.Join(input, " "),
	})
	l(log.Log()).Info("Start")
	cmd := exec.Command(input[0], input[1:]...)
	stdout, errOUT := cmd.StdoutPipe()
	if errOUT != nil {
		l(log.LogE(errOUT)).Warning("Impossible to obtain the STDOUT pipe")
		return nil
	}
	stderr, errERR := cmd.StderrPipe()
	if errERR != nil {
		l(log.LogE(errERR)).Warning("Impossible to obtain the STDERR pipe")
		return nil
	}

	return &execCmd{cmd: cmd, err: stderr, out: stdout}
}

func (e *execCmd) Start() error {
	l := log.Decorate(map[string]string{
		"Action": "executing command",
	})
	if e == nil {
		err := fmt.Errorf("Use of nil execCmd")
		l(log.LogE(err)).Error("Call start with nil cmd, maybe error in the constructor")
		return err
	}

	err := e.cmd.Start()
	if err != nil {
		l(log.LogE(err)).Error("Error in starting the command")
		return err
	}

	slurpOut, errOUT := ioutil.ReadAll(e.out)
	if errOUT != nil {
		l(log.LogE(errOUT)).Warning("Impossible to read the STDOUT")
		return err
	}
	slurpErr, errERR := ioutil.ReadAll(e.err)
	if errERR != nil {
		l(log.LogE(errERR)).Warning("Impossible to read the STDERR")
		return err
	}

	err = e.cmd.Wait()
	if err != nil {
		l(log.LogE(err)).Error("Error in executing the command")
		l(log.Log()).WithFields(logrus.Fields{"pipe": "STDOUT"}).Info(string(slurpOut))
		l(log.Log()).WithFields(logrus.Fields{"pipe": "STDERR"}).Info(string(slurpErr))
		return err
	}
	return nil
}

type RemoteTar interface {
	Download()
	Error() error
	TemporaryLocation() string
	Clean() error
}

type IngestableTar interface {
	RemoteTar
	CVMFSLocation() string
}

type IngestionResult interface {
	IngestedTar() *IngestableTar
	Error() error
}

type SimpleIngestionResult struct {
	ingestableTar *IngestableTar
	err           error
}

func (res SimpleIngestionResult) IngestedTar() *IngestableTar {
	return res.ingestableTar
}

func (res SimpleIngestionResult) Error() error {
	return res.err
}

func CVMFSLoop(CVMFSRepo string) (chan IngestableTar, chan IngestionResult) {
	input := make(chan IngestableTar, 10)
	results := make(chan IngestionResult, 10)

	go func() {
		defer close(input)
		defer close(results)
		for tarToIngest := range input {
			ingestionResult := IngestTar(CVMFSRepo, tarToIngest)
			results <- ingestionResult
		}
	}()

	return input, results
}

func IngestTar(CVMFSRepo string, tar IngestableTar) IngestionResult {
	result := SimpleIngestionResult{ingestableTar: &tar, err: nil}
	err := ExecCommand("cvmfs_server", "ingest",
		"-t", tar.TemporaryLocation(),
		"-b", tar.CVMFSLocation(),
		CVMFSRepo).Start()
	result.err = err
	return result
}

type Repo struct {
	Name string
	Lock sync.Mutex
}

func NewRepo(name string) Repo {
	m := &sync.Mutex{}
	return Repo{name, *m}
}
