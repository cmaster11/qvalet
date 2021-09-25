package pkg

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"

	"gotoexec/pkg/utils"

	"github.com/Masterminds/goutils"
	"github.com/beyondstorage/go-storage/v4/types"
	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CompiledListener struct {
	config *ListenerConfig
	log    logrus.FieldLogger

	route string

	// If this is an error handler, this would be the original route
	sourceRoute string

	isErrorHandler bool

	tplCmd   *Template
	tplArgs  []*Template
	tplEnv   map[string]*Template
	tplFiles map[string]*Template

	storager      types.Storager
	storagePrefix string

	errorHandler *CompiledListener

	// Maps fixed file names to execution-time file names
	tplTmpFileNames              map[string]interface{}
	tplTmpFileNamesOriginalPaths map[string]interface{}

	plugins []Plugin
}

func (listener *CompiledListener) Route() string {
	return listener.route
}

func (listener *CompiledListener) Logger() logrus.FieldLogger {
	return logrus.WithField("listener", listener.Route())
}

const funcMapKeyGTE = "gte"

func (listener *CompiledListener) clone() *CompiledListener {
	tplCmdClone, _ := listener.tplCmd.Clone()
	var tplArgsClones []*Template
	for _, tpl := range listener.tplArgs {
		clone, _ := tpl.Clone()
		tplArgsClones = append(tplArgsClones, clone)
	}
	tplEnvClones := make(map[string]*Template)
	for key, tpl := range listener.tplEnv {
		clone, _ := tpl.Clone()
		tplEnvClones[key] = clone
	}
	tplFilesClones := make(map[string]*Template)
	for key, tpl := range listener.tplFiles {
		clone, _ := tpl.Clone()
		tplFilesClones[key] = clone
	}

	var errorHandler *CompiledListener
	if listener.errorHandler != nil {
		errorHandler = listener.errorHandler.clone()
	}

	newListener := &CompiledListener{
		listener.config,
		listener.log,
		listener.route,
		listener.sourceRoute,
		listener.isErrorHandler,
		tplCmdClone,
		tplArgsClones,
		tplEnvClones,
		tplFilesClones,
		listener.storager,
		listener.storagePrefix,
		errorHandler,
		// On clone, generate a new execution-time temporary files map
		map[string]interface{}{},
		map[string]interface{}{},
		[]Plugin{},
	}

	var newPlugins []Plugin
	for _, p := range listener.plugins {
		newPlugins = append(newPlugins, p.Clone(newListener))
	}
	newListener.plugins = newPlugins

	// Replace the gte function in all cloned templates
	funcMap := newListener.TplFuncMap()
	newListener.tplCmd.Funcs(funcMap)
	for _, tpl := range newListener.tplArgs {
		tpl.Funcs(funcMap)
	}
	for _, tpl := range newListener.tplEnv {
		tpl.Funcs(funcMap)
	}
	for _, tpl := range newListener.tplFiles {
		tpl.Funcs(funcMap)
	}

	return newListener
}

var regexClearPath = regexp.MustCompile(`/+`)

func compileListener(
	defaults *ListenerConfig,
	listenerConfig *ListenerConfig,
	route string,
	isErrorHandler bool,
	storageCache *sync.Map,
) *CompiledListener {
	sourceRoute := route
	if isErrorHandler {
		route = fmt.Sprintf("%s-on-error", route)
	}

	log := logrus.WithField("listener", route)

	listenerConfig, err := MergeListenerConfig(defaults, listenerConfig)
	if err != nil {
		log.WithError(err).Fatal("failed to merge listener config")
	}

	if err := utils.Validate.Struct(listenerConfig); err != nil {
		log.WithError(err).Fatal("failed to validate listener config")
	}

	if isErrorHandler {
		// Error handlers do NOT need certain features, so disable them
		listenerConfig.Auth = nil
		listenerConfig.ErrorHandler = nil
		listenerConfig.Trigger = nil
	}

	listener := &CompiledListener{
		config:         listenerConfig,
		log:            log,
		route:          route,
		sourceRoute:    sourceRoute,
		isErrorHandler: isErrorHandler,
	}

	if listenerConfig.ErrorHandler != nil {
		listener.errorHandler = compileListener(defaults, listenerConfig.ErrorHandler, route, true, storageCache)
	}

	tplFuncs := listener.TplFuncMap()

	// Creates a unique tmp directory where to store the files
	{
		tplFiles := make(map[string]*Template)
		for key, content := range listener.config.Files {
			filePath := key

			tpl, err := ParseTemplate(fmt.Sprintf("files-%s", key), content, tplFuncs)
			if err != nil {
				log.WithError(err).WithField("file", key).WithField("template", tpl).Fatal("failed to parse listener file template")
			}
			tplFiles[filePath] = tpl
		}
		listener.tplFiles = tplFiles
	}

	{
		tplCmd, err := ParseTemplate(route, listenerConfig.Command, tplFuncs)
		if err != nil {
			log.WithError(err).WithField("template", listenerConfig.Command).Fatal("failed to parse listener command template")
		}
		listener.tplCmd = tplCmd
	}

	{
		var tplArgs []*Template
		for idx, str := range listenerConfig.Args {
			tpl, err := ParseTemplate(fmt.Sprintf("%s-%d", route, idx), str, tplFuncs)
			if err != nil {
				log.WithError(err).WithField("template", tpl).Fatal("failed to parse listener args template")
			}
			tplArgs = append(tplArgs, tpl)
		}
		listener.tplArgs = tplArgs
	}

	{
		tplEnv := make(map[string]*Template)
		for key, content := range listener.config.Env {
			tpl, err := ParseTemplate(fmt.Sprintf("env-%s", key), content, tplFuncs)
			if err != nil {
				log.WithError(err).WithField("file", key).WithField("template", tpl).Fatal("failed to parse listener env template")
			}
			tplEnv[key] = tpl
		}
		listener.tplEnv = tplEnv
	}

	// If storage is defined, we need to initialize the storager
	if listenerConfig.Storage != nil && len(listenerConfig.Storage.Store) > 0 {
		// Re-use already-found instances
		if storageCache != nil {
			if storager, found := storageCache.Load(listenerConfig.Storage.Conn); found {
				listener.storager = storager.(types.Storager)
				goto afterStorage
			}
		}

		storager, err := GetStoragerFromString(listenerConfig.Storage.Conn)
		if err != nil {
			log := log.WithError(err)
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				log = log.WithField("conn", listenerConfig.Storage.Conn)
			}
			log.Fatal("failed to initialize storage")
		}

		// Check that we can write there
		{
			routePrefix := regexListenerRouteCleaner.ReplaceAllString(listener.route, "_")
			nowNano := time.Now().UnixNano()
			rand, _ := goutils.RandomAlphaNumeric(8)
			p := fmt.Sprintf("%s-testwrite-%d-%s", routePrefix, nowNano, rand)

			b := []byte(fmt.Sprintf("%d", nowNano))
			_, err = storager.Write(p, bytes.NewBuffer(b), int64(len(b)))
			if err != nil {
				listener.log.WithError(err).Fatal("failed to check if storage is writable")
			}

			listener.log.WithField("path", p).Debug("written check writable file")

			// Clean up if possible
			if err := storager.Delete(p); err != nil {
				listener.log.WithError(err).Debug("failed to remove check writable storage file")
			}

		}

		if storageCache != nil {
			storageCache.Store(listenerConfig.Storage.Conn, storager)
		}

		listener.storager = storager
	}
afterStorage:

	// Group all plugins together
	var listenerPlugins []Plugin
	for _, pluginsEntry := range listenerConfig.Plugins {
		list, err := pluginsEntry.ToPluginList(listener)
		if err != nil {
			listener.log.WithError(err).Fatal("failed to initialize plugins list")
		}

		listenerPlugins = append(listenerPlugins, list...)
	}
	listener.plugins = listenerPlugins

	return listener
}

func (listener *CompiledListener) tplGTE() map[string]interface{} {
	return map[string]interface{}{
		"files": listener.tplTmpFileNames,
	}
}
func (listener *CompiledListener) TplFuncMap() template.FuncMap {
	return template.FuncMap{
		// Added here to make tpls parse, but will be overwritten on clone
		funcMapKeyGTE: listener.tplGTE,
	}
}

// @formatter:off
/// [exec-command-result]
type ExecCommandResult struct {
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
	Output  string   `json:"output,omitempty"`
}

/// [exec-command-result]
// @formatter:on

func (listener *CompiledListener) ExecCommand(args map[string]interface{}, toStore map[string]interface{}) (*ExecCommandResult, error) {
	// Execute all pre-hooks
	for _, plugin := range listener.plugins {
		if plugin, ok := plugin.(PluginHookPreExecute); ok {
			_args, err := plugin.HookPreExecute(args)
			if err != nil {
				return nil, errors.WithMessage(err, "failed to execute pre-hook plugin")
			}
			args = _args
		}
	}

	if listener.storager != nil && listener.config.Storage.StoreArgs() {
		toStore["args"] = args
	}

	/*
		Create a new instance of the listener, to handle temporary files.

		On every new run, we store files in different temporary folders, and we need to populate
		the "files" map of the template with different values, which means pointing the "gte" function
		to a different listener!
	*/
	log := listener.log

	if listener.config.LogArgs() {
		log = log.WithField("args", args)
	}

	if listener.config.Trigger != nil {
		// The listener has a trigger condition, so evaluate it
		isTrue, err := listener.config.Trigger.IsTrue(args)
		if err != nil {
			err := errors.WithMessage(err, "failed to evaluate listener trigger condition")
			log.WithError(err).Error("error")
			return nil, err
		}

		if !isTrue {
			// All good, do nothing
			return &ExecCommandResult{
				Output: "not triggered",
			}, nil
		}
	}

	if err := listener.processFiles(args); err != nil {
		err := errors.WithMessage(err, "failed to process temporary files")
		log.WithError(err).Error("error")
		return nil, err
	}

	var cmdStr string
	{
		out, err := listener.tplCmd.Execute(args)
		if err != nil {
			err := errors.WithMessage(err, "failed to execute command template")
			log.WithError(err).Error("error")
			return nil, err
		}
		cmdStr = out
	}

	var cmdArgs []string
	for _, tpl := range listener.tplArgs {
		out, err := tpl.Execute(args)
		if err != nil {
			err := errors.WithMessagef(err, "failed to execute args template %s", tpl.Name())
			log.WithError(err).Error("error")
			return nil, err
		}
		cmdArgs = append(cmdArgs, out)
	}

	var cmdEnv []string
	for key, tpl := range listener.tplEnv {
		out, err := tpl.Execute(args)
		if err != nil {
			err := errors.WithMessagef(err, "failed to execute env template %s", tpl.Name())
			log.WithError(err).Error("error")
			return nil, err
		}
		// For env vars, we need to remove any new lines
		out = strings.ReplaceAll(out, "\n", "")
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", key, out))
	}

	for cleanPath, realPath := range listener.tplTmpFileNames {
		cmdEnv = append(cmdEnv, fmt.Sprintf("GTE_FILES_%s=%s", cleanPath, realPath))
	}

	{
		logCommandFields := map[string]interface{}{}
		if listener.config.LogCommand() {
			logCommandFields["command"] = cmdStr
			logCommandFields["args"] = cmdArgs
		}
		if listener.config.LogEnv() {
			logCommandFields["env"] = cmdEnv
		}
		if len(logCommandFields) > 0 {
			log = log.WithFields(logrus.Fields{
				"command": logCommandFields,
			})
		}
	}

	toReturn := &ExecCommandResult{}

	if listener.config.ReturnCommand() {
		toReturn.Command = cmdStr
		toReturn.Args = cmdArgs
	}
	if listener.config.ReturnEnv() {
		toReturn.Env = cmdEnv
	}

	cmd := exec.Command(cmdStr, cmdArgs...)
	cmd.Env = os.Environ()

	for _, env := range cmdEnv {
		cmd.Env = append(cmd.Env, env)
	}

	if listener.storager != nil {
		storeCommandFields := map[string]interface{}{}
		if listener.config.Storage.StoreCommand() {
			storeCommandFields["command"] = cmdStr
			storeCommandFields["args"] = cmdArgs
		}
		if listener.config.Storage.StoreEnv() {
			storeCommandFields["env"] = cmdEnv
		}

		if len(storeCommandFields) > 0 {
			toStore["command"] = storeCommandFields
		}
	}

	out, err := cmd.CombinedOutput()
	outStr := string(out)

	if listener.storager != nil && listener.config.Storage.StoreOutput() {
		toStore["output"] = outStr
	}

	if listener.config.ReturnOutput() {
		toReturn.Output = outStr
	}

	if err != nil {
		if listener.storager != nil && listener.config.Storage.StoreOutput() {
			toStore["error"] = err.Error()
		}

		msg := "failed to execute command"
		err := errors.WithMessage(err, msg)

		log := log
		if listener.config.LogOutput() {
			log = log.WithField("output", outStr)
		}

		log.WithError(err).Error("error")

		return toReturn, err
	}

	if listener.config.LogOutput() {
		log = log.WithField("output", outStr)
	}

	if !listener.config.ReturnOutput() {
		toReturn.Output = "success"
	}

	log.Info("command executed")

	return toReturn, nil
}

func (listener *CompiledListener) HandleRequest(c *gin.Context, args map[string]interface{}) (bool, *ListenerResponse, error) {
	// Keep track of what to store
	toStore := make(map[string]interface{})

	l := listener.clone()

	out, err := l.ExecCommand(args, toStore)
	defer l.cleanTemporaryFiles()
	if err != nil {
		err := errors.WithMessagef(err, "failed to execute listener %s", l.route)
		response := &ListenerResponse{
			ExecCommandResult: out,
			Error:             stringPtr(err.Error()),
		}

		var errorHandlerResult *ListenerResponse
		if l.errorHandler != nil {
			errorHandler := l.errorHandler
			errorHandlerResult = &ListenerResponse{}

			toStoreOnError := make(map[string]interface{})

			// Trigger a command on error
			onErrorArgs := map[string]interface{}{
				"route":  l.route,
				"error":  err.Error(),
				"output": out,
				"args":   args,
			}

			if errorHandler.storager != nil && errorHandler.config.Storage.StoreArgs() {
				toStoreOnError["args"] = args
			}

			errorHandlerExecCommandResult, err := errorHandler.ExecCommand(onErrorArgs, toStoreOnError)
			defer errorHandler.cleanTemporaryFiles()
			errorHandlerResult.ExecCommandResult = errorHandlerExecCommandResult
			if err != nil {
				errorHandlerResult.Error = stringPtr(err.Error())
				errorHandler.log.WithError(err).Error("failed to execute error listener")
			} else {
				errorHandler.log.Info("executed error listener")
			}

			if errorHandler.storager != nil && len(toStoreOnError) > 0 {
				if entry := storePayload(
					errorHandler,
					toStoreOnError,
				); entry != nil {
					if errorHandler.config.ReturnStorage() {
						errorHandlerResult.Storage = entry
					}
				}
			}

			toStore["errorHandler"] = toStoreOnError
			if l.storager != nil && len(toStore) > 0 {
				if entry := storePayload(
					l,
					toStore,
				); entry != nil {
					if l.config.ReturnStorage() {
						response.Storage = entry
					}
				}
			}

			response.ErrorHandlerResult = errorHandlerResult
		}

		return false, response, err
	}

	response := &ListenerResponse{
		ExecCommandResult: out,
	}

	if l.storager != nil && len(toStore) > 0 {
		if entry := storePayload(
			listener,
			toStore,
		); entry != nil {
			if l.config.ReturnStorage() {
				response.Storage = entry
			}
		}
	}

	for _, plugin := range l.plugins {
		if p, ok := plugin.(PluginHookOutput); ok {
			handled, err := p.HookOutput(c, args, response)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errors.WithMessage(err, "failed to process output via plugin"))
				return true, response, err
			}
			if handled {
				return true, response, nil
			}
		}
	}

	return false, response, nil
}

var regexReplaceTemporaryFileName = regexp.MustCompile(`\W`)

// processFiles stores files defined in the "files" listener config entry in the right place
func (listener *CompiledListener) processFiles(args map[string]interface{}) error {
	log := listener.log

	filesDir := ""
	tplTmpFileNames := make(map[string]interface{})
	tplTmpFileNamesOriginalPaths := make(map[string]interface{})
	for key, tpl := range listener.tplFiles {
		log := log.WithField("file", key)

		originalFilePath := key
		realFilePath := originalFilePath
		if !path.IsAbs(originalFilePath) {
			if filesDir == "" {
				_filesDir, err := os.MkdirTemp("", "gte-")
				if err != nil {
					err := errors.WithMessage(err, "failed to create temporary files directory")
					log.WithError(err).Error("error")
					return err
				}
				filesDir = _filesDir
			}
			realFilePath = filepath.Join(filesDir, originalFilePath)
		}
		cleanFileName := regexReplaceTemporaryFileName.ReplaceAllString(key, "_")
		tplTmpFileNames[cleanFileName] = realFilePath
		tplTmpFileNamesOriginalPaths[cleanFileName] = originalFilePath

		out, err := tpl.Execute(args)
		if err != nil {
			err := errors.WithMessage(err, "failed to execute file template")
			log.WithError(err).Error("error")
			return err
		}

		if err := os.WriteFile(realFilePath, []byte(out), 0777); err != nil {
			err := errors.WithMessage(err, "failed to write file template")
			log.WithError(err).Error("error")
			return err
		}

		log.Debugf("written temporary file %s at %s", originalFilePath, realFilePath)
	}
	listener.tplTmpFileNames = tplTmpFileNames
	listener.tplTmpFileNamesOriginalPaths = tplTmpFileNamesOriginalPaths

	return nil
}

func (listener *CompiledListener) cleanTemporaryFiles() {
	log := listener.log

	for key, filePathIntf := range listener.tplTmpFileNamesOriginalPaths {
		filePath := filePathIntf.(string)

		// Do NOT remove files with absolute paths
		if path.IsAbs(filePath) {
			continue
		}

		realPath := listener.tplTmpFileNames[key].(string)

		log := log.WithField("file", realPath)

		if err := os.Remove(realPath); err != nil {
			err := errors.WithMessage(err, "failed to remove file template")
			log.WithError(err).Error("error")
		} else {
			log.Debugf("removed temporary file %s at %s", filePath, realPath)
		}
	}
}
