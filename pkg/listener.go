package pkg

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"text/template"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CompiledListener struct {
	config *ListenerConfig
	log    logrus.FieldLogger

	route string

	tplCmd   *template.Template
	tplArgs  []*template.Template
	tplFiles map[string]*template.Template

	// Maps fixed file names to execution-time file names
	tplTmpFileNames map[string]interface{}
}

const funcMapKeyGTE = "gte"

func (listener *CompiledListener) clone() *CompiledListener {
	tplCmdClone, _ := listener.tplCmd.Clone()
	var tplArgsClones []*template.Template
	for _, tpl := range listener.tplArgs {
		clone, _ := tpl.Clone()
		tplArgsClones = append(tplArgsClones, clone)
	}
	tplFilesClones := make(map[string]*template.Template)
	for key, tpl := range listener.tplFiles {
		clone, _ := tpl.Clone()
		tplFilesClones[key] = clone
	}

	newListener := &CompiledListener{
		listener.config,
		listener.log,
		listener.route,
		tplCmdClone,
		tplArgsClones,
		tplFilesClones,
		// On clone, generate a new execution-time temporary files map
		map[string]interface{}{},
	}

	funcMap := template.FuncMap{
		funcMapKeyGTE: newListener.tplGTE,
	}

	// Replace the gte function in all cloned templates
	newListener.tplCmd.Funcs(funcMap)
	for _, tpl := range newListener.tplArgs {
		tpl.Funcs(funcMap)
	}
	for _, tpl := range newListener.tplFiles {
		tpl.Funcs(funcMap)
	}

	return newListener
}

func (gte *GoToExec) compileListener(listenerConfig *ListenerConfig, route string) *CompiledListener {
	log := logrus.WithField("listener", route)

	listener := &CompiledListener{
		config: listenerConfig,
		log:    log,
		route:  route,
	}

	tplFuncs := GetTPLFuncsMap()

	// Added here to make tpls parse, but will be overwritten on clone
	tplFuncs[funcMapKeyGTE] = listener.tplGTE

	// Creates a unique tmp directory where to store the files
	{
		tplFiles := make(map[string]*template.Template)
		for key, content := range listener.config.Files {
			filePath := key

			tpl, err := template.New(fmt.Sprintf("files-%s", key)).Funcs(tplFuncs).Parse(content)
			if err != nil {
				log.WithError(err).WithField("file", key).WithField("template", tpl).Fatal("failed to parse listener file template")
			}
			tplFiles[filePath] = tpl
		}
		listener.tplFiles = tplFiles
	}

	{
		tplCmd, err := template.New(route).Funcs(tplFuncs).Parse(listenerConfig.Command)
		if err != nil {
			log.WithError(err).WithField("template", listenerConfig.Command).Fatal("failed to parse listener command template")
		}
		listener.tplCmd = tplCmd
	}

	{
		var tplArgs []*template.Template
		for idx, str := range listenerConfig.Args {
			tpl, err := template.New(fmt.Sprintf("%s-%d", route, idx)).Funcs(tplFuncs).Parse(str)
			if err != nil {
				log.WithError(err).WithField("template", tpl).Fatal("failed to parse listener args template")
			}
			tplArgs = append(tplArgs, tpl)
		}
		listener.tplArgs = tplArgs
	}
	return listener
}

func (listener *CompiledListener) tplGTE() map[string]interface{} {
	return map[string]interface{}{
		"files": listener.tplTmpFileNames,
	}
}

func (listener *CompiledListener) ExecCommand(args map[string]interface{}) (string, error) {
	/*
		Create a new instance of the listener, to handle temporary files.

		On every new run, we store files in different temporary folders, and we need to populate
		the "files" map of the template with different values, which means pointing the "gte" function
		to a different listener!
	*/
	l := listener.clone()

	log := l.log

	if l.config.LogArgs {
		log = log.WithField("args", args)
	}

	if err := l.processTemporaryFiles(args); err != nil {
		err := errors.WithMessage(err, "failed to process temporary files")
		log.WithError(err).Error("error")
		return "", err
	}
	defer l.cleanTemporaryFiles()

	var tplCmdOutput bytes.Buffer
	if err := l.tplCmd.Execute(&tplCmdOutput, args); err != nil {
		err := errors.WithMessage(err, "failed to execute command template")
		log.WithError(err).Error("error")
		return "", err
	}

	cmdStr := tplCmdOutput.String()

	var cmdArgs []string
	for _, tpl := range l.tplArgs {
		var tplArgOutput bytes.Buffer
		if err := tpl.Execute(&tplArgOutput, args); err != nil {
			err := errors.WithMessagef(err, "failed to execute args template %s", tpl.Name())
			log.WithError(err).Error("error")
			return "", err
		}
		arg := tplArgOutput.String()
		cmdArgs = append(cmdArgs, arg)
	}

	if l.config.LogCommand {
		log = log.WithFields(logrus.Fields{
			"command":     cmdStr,
			"commandArgs": cmdArgs,
		})
	}

	cmd := exec.Command(cmdStr, cmdArgs...)
	cmd.Env = os.Environ()

	for cleanPath, realPath := range l.tplTmpFileNames {
		cmd.Env = append(cmd.Env, fmt.Sprintf("GTE_FILES_%s=%s", cleanPath, realPath))
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := "failed to execute command"

		if l.config.ReturnOutput {
			msg += ": " + string(out)
		}

		err := errors.WithMessage(err, msg)

		log := log
		if l.config.LogOutput {
			log = log.WithField("output", string(out))
		}

		log.WithError(err).Error("error")
		return "", err
	}

	if l.config.LogOutput {
		log = log.WithField("output", string(out))
	}

	log.Info("command executed")

	if l.config.ReturnOutput {
		return string(out), nil
	}

	return "success", nil
}

var regexReplaceTemporaryFileName = regexp.MustCompile(`\W`)

// processTemporaryFiles stores temporary files defined in the "files" listener config entry in the right place
func (listener *CompiledListener) processTemporaryFiles(args map[string]interface{}) error {
	log := listener.log

	filesDir := ""
	tplTmpFileNames := make(map[string]interface{})
	for key, tpl := range listener.tplFiles {
		log := log.WithField("file", key)

		filePath := key
		if !path.IsAbs(filePath) {
			if filesDir == "" {
				_filesDir, err := os.MkdirTemp("", "gte-")
				if err != nil {
					err := errors.WithMessage(err, "failed to create temporary files directory")
					log.WithError(err).Error("error")
					return err
				}
				filesDir = _filesDir
			}
			filePath = filepath.Join(filesDir, filePath)
		}
		cleanFileName := regexReplaceTemporaryFileName.ReplaceAllString(key, "_")
		tplTmpFileNames[cleanFileName] = filePath

		var tplFileOutput bytes.Buffer
		if err := tpl.Execute(&tplFileOutput, args); err != nil {
			err := errors.WithMessage(err, "failed to execute file template")
			log.WithError(err).Error("error")
			return err
		}

		if err := os.WriteFile(filePath, tplFileOutput.Bytes(), 0777); err != nil {
			err := errors.WithMessage(err, "failed to write file template")
			log.WithError(err).Error("error")
			return err
		}

		log.Debugf("written temporary file %s", filePath)
	}
	listener.tplTmpFileNames = tplTmpFileNames

	return nil
}

func (listener *CompiledListener) cleanTemporaryFiles() {
	log := listener.log

	for _, filePath := range listener.tplTmpFileNames {
		log := log.WithField("file", filePath)

		if err := os.Remove(filePath.(string)); err != nil {
			err := errors.WithMessage(err, "failed to remove file template")
			log.WithError(err).Error("error")
		}

		log.Debugf("removed temporary file %s", filePath)
	}
}
