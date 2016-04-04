package env

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"edgeg.io/gtm/scm"
)

var (
	ErrNotInitialized = errors.New("Git Time Metric is not initialized")
	ErrFileNotFound   = errors.New("File does not exist")
)

var (
	NoteNameSpace string = "gtm-data"
	GTMDirectory  string = ".gtm"
	GitHooks             = map[string]string{
		"pre-push":    "git push --no-verify origin refs/notes/gtm-data",
		"post-commit": "gtm commit --dry-run=false"}
	GitConfig = map[string]string{
		"remote.origin.fetch": "+refs/notes/gtm-data:refs/notes/gtm-data",
		"notes.rewriteref":    "refs/notes/gtm-data"}
	GitIgnore string = ".gtm/"
)

const InitMsgTpl string = `
Git Time Metric has been initialized
------------------------------------
{{ range $hook, $command := .GitHooks -}}
{{$hook}}: "{{$command}}"
{{end -}}
{{ range $key, $val := .GitConfig -}}
{{$key}}: "{{$val}}"
{{end -}}
gitignore: "{{.GitIgnore}}"
`

var Now = func() time.Time { return time.Now() }

func Initialize() (string, error) {
	var fp string

	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	fp = filepath.Join(wd, ".git")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return "", fmt.Errorf(
			"Unable to intialize Git Time Metric, Git repository not found in %s", wd)
	}

	fp = filepath.Join(wd, GTMDirectory)
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		if err := os.MkdirAll(fp, 0700); err != nil {
			return "", err
		}
	}

	if err := scm.GitInitHooks(GitHooks); err != nil {
		return "", err
	}

	if err := scm.GitConfig(GitConfig); err != nil {
		return "", err
	}

	if err := scm.GitIgnore(GitIgnore); err != nil {
		return "", err
	}

	b := new(bytes.Buffer)
	t := template.Must(template.New("msg").Parse(InitMsgTpl))
	err = t.Execute(b,
		struct {
			GTMPath   string
			GitHooks  map[string]string
			GitConfig map[string]string
			GitIgnore string
		}{
			fp,
			GitHooks,
			GitConfig,
			GitIgnore})

	return b.String(), nil
}

// The Paths function returns the git repository root path and the gtm path within the root.
// If the path is not blank, it's used as the current working directory when retrieving the root path.
//
// Note - the function is declared as a variable to allow for mocking during testing.
//
var Paths = func(path ...string) (string, string, error) {
	p := ""
	if len(path) > 0 {
		p = path[0]
	}
	rootPath, err := scm.GitRootPath(p)
	if err != nil {
		return "", "", ErrNotInitialized
	}

	gtmPath := filepath.Join(rootPath, GTMDirectory)
	if _, err := os.Stat(gtmPath); os.IsNotExist(err) {
		return "", "", ErrNotInitialized
	}
	return rootPath, gtmPath, nil
}

func LogToGTM(v ...interface{}) error {
	_, gtmPath, err := Paths()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(gtmPath, "gtm.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("error opening log file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	log.Println(v)
	return nil
}
