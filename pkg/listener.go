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
	// The id will be filled at runtime
	id string

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
	tplTmpFileNamesOriginalPaths map[string]string

	plugins []PluginInterface

	dbWrapper *BunDbWrapper
}

func (listener *CompiledListener) Plugins() []PluginInterface {
	return listener.plugins
}

func (listener *CompiledListener) Logger() logrus.FieldLogger {
	return logrus.WithField("listener", listener.route)
}

func (listener *CompiledListener) SetId(value string) {
	listener.id = value
}

const funcMapKeyGTE = "gte"

func (listener *CompiledListener) clone() (*CompiledListener, error) {
	newListener := &CompiledListener{
		listener.id,
		listener.config,
		listener.log,
		listener.route,
		listener.sourceRoute,
		listener.isErrorHandler,
		nil,
		nil,
		nil,
		nil,
		listener.storager,
		listener.storagePrefix,
		nil,
		// On clone, generate a new execution-time temporary files map
		map[string]interface{}{},
		map[string]string{},
		[]PluginInterface{},
		listener.dbWrapper,
	}

	tplCmdClone, err := listener.tplCmd.CloneForListener(newListener)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to clone command template")
	}
	newListener.tplCmd = tplCmdClone

	var tplArgsClones []*Template
	for _, tpl := range listener.tplArgs {
		clone, err := tpl.CloneForListener(newListener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to clone arg template")
		}
		tplArgsClones = append(tplArgsClones, clone)
	}
	newListener.tplArgs = tplArgsClones

	tplEnvClones := make(map[string]*Template)
	for key, tpl := range listener.tplEnv {
		clone, err := tpl.CloneForListener(newListener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to clone env template")
		}
		tplEnvClones[key] = clone
	}
	newListener.tplEnv = tplEnvClones

	tplFilesClones := make(map[string]*Template)
	for key, tpl := range listener.tplFiles {
		clone, err := tpl.CloneForListener(newListener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to clone file template")
		}
		tplFilesClones[key] = clone
	}
	newListener.tplFiles = tplFilesClones

	if listener.errorHandler != nil {
		errorHandler, err := listener.errorHandler.clone()
		if err != nil {
			return nil, errors.WithMessage(err, "failed to clone error handler listener")
		}
		newListener.errorHandler = errorHandler
	}

	var newPlugins []PluginInterface
	for _, p := range listener.plugins {
		clone, err := p.Clone(newListener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to clone plugin")
		}
		newPlugins = append(newPlugins, clone)
	}
	newListener.plugins = newPlugins

	return newListener, nil
}

func compileListener(
	defaults *ListenerConfig,
	listenerConfig *ListenerConfig,
	route string,
	isErrorHandler bool,
	storageCache *sync.Map,
) (*CompiledListener, error) {
	sourceRoute := route
	if isErrorHandler {
		route = fmt.Sprintf("%s-on-error", route)
	}

	log := logrus.WithField("listener", route)

	listenerConfig, err := MergeListenerConfig(defaults, listenerConfig)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to merge listener config")
	}

	if err := utils.Validate.Struct(listenerConfig); err != nil {
		return nil, errors.WithMessage(err, "failed to validate listener config")
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

		tplCmd:   listenerConfig.Command,
		tplArgs:  listenerConfig.Args,
		tplEnv:   listenerConfig.Env,
		tplFiles: listenerConfig.Files,
	}

	if listenerConfig.ErrorHandler != nil {
		errorHandler, err := compileListener(defaults, listenerConfig.ErrorHandler, route, true, storageCache)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to compile error handler listener")
		}
		listener.errorHandler = errorHandler
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
			// Suffix for debugging purposes
			suffix := ""
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				suffix = fmt.Sprintf(" for conn %s", listenerConfig.Storage.Conn)
			}
			return nil, errors.WithMessagef(err, "failed to initialize storage%s", suffix)
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
				return nil, errors.WithMessage(err, "failed to check if storage is writable")
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
	var listenerPlugins []PluginInterface
	var dbRequiredForPlugins []PluginConfigNeedsDb
	for _, pluginsEntry := range listenerConfig.Plugins {
		list, err := pluginsEntry.ToPluginList(listener)
		if err != nil {
			return nil, errors.WithMessage(err, "failed to initialize plugins list")
		}

		for _, p := range list {
			if p, ok := p.(PluginConfigNeedsDb); ok && p.NeedsDb() {
				dbRequiredForPlugins = append(dbRequiredForPlugins, p)
			}
		}

		listenerPlugins = append(listenerPlugins, list...)
	}
	listener.plugins = listenerPlugins

	// If a database is defined and is required, connect!
	if len(dbRequiredForPlugins) > 0 {
		if listenerConfig.Database == nil {
			return nil, errors.WithMessagef(err, "database is required for plugins %v to work", dbRequiredForPlugins)
		}

		db, err := NewDB(listenerConfig.Database)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed to initialize database at %s", listenerConfig.Database.ParsedLogSafeDSN())
		}

		// Perform any pending migrations
		for _, p := range dbRequiredForPlugins {
			if err := db.ApplyMigrations(p); err != nil {
				return nil, errors.WithMessagef(err, "failed to apply migrations for plugin %s", p.Id())
			}
		}

		listener.dbWrapper = db
	}

	return listener, nil
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
	Command string   `json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string `json:"args,omitempty" yaml:"args,omitempty"`
	Env     []string `json:"env,omitempty" yaml:"env,omitempty"`
	Output  string   `json:"output,omitempty" yaml:"output,omitempty"`
}

/// [exec-command-result]
// @formatter:on

func (listener *CompiledListener) ExecCommand(args map[string]interface{}, toStore map[string]interface{}) (*ExecCommandResult, error) {
	log := listener.log

	preparedExecutionResult, handledResult, err := listener.prepareExecution(args, toStore)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to prepare command execution")
	}

	if handledResult != nil {
		return handledResult, nil
	}

	cmdStr := preparedExecutionResult.Command
	cmdArgs := preparedExecutionResult.Args
	cmdEnv := preparedExecutionResult.Env

	if listener.config.LogArgs() {
		log = log.WithField("args", args)
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

		err := errors.WithMessage(err, "failed to execute command")

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

type preparedExecutionResult struct {
	Command string   `json:"command,omitempty" yaml:"command,omitempty"`
	Args    []string `json:"args,omitempty" yaml:"args,omitempty"`
	Env     []string `json:"env,omitempty" yaml:"env,omitempty"`
}

func (listener *CompiledListener) prepareExecution(args map[string]interface{}, toStore map[string]interface{}) (*preparedExecutionResult, *ExecCommandResult, error) {
	// Execute all pre-hooks
	for _, plugin := range listener.plugins {
		if plugin, ok := plugin.(PluginHookPreExecute); ok {
			_args, err := plugin.HookPreExecute(args)
			if err != nil {
				return nil, nil, errors.WithMessage(err, "failed to execute pre-hook plugin")
			}
			args = _args
		}
	}

	if listener.storager != nil && listener.config.Storage.StoreArgs() {
		toStore["args"] = args
	}

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
			return nil, nil, err
		}

		if !isTrue {
			// All good, do nothing
			return nil, &ExecCommandResult{
				Output: "not triggered",
			}, nil
		}
	}

	if err := listener.processFiles(args); err != nil {
		err := errors.WithMessage(err, "failed to process temporary files")
		log.WithError(err).Error("error")
		return nil, nil, err
	}

	var cmdStr string
	{
		out, err := listener.tplCmd.Execute(args)
		if err != nil {
			err := errors.WithMessage(err, "failed to execute command template")
			log.WithError(err).Error("error")
			return nil, nil, err
		}
		cmdStr = out
	}

	var cmdArgs []string
	for _, tpl := range listener.tplArgs {
		out, err := tpl.Execute(args)
		if err != nil {
			err := errors.WithMessagef(err, "failed to execute args template %s", tpl.Name())
			log.WithError(err).Error("error")
			return nil, nil, err
		}
		cmdArgs = append(cmdArgs, out)
	}

	var cmdEnv []string
	for key, tpl := range listener.tplEnv {
		out, err := tpl.Execute(args)
		if err != nil {
			err := errors.WithMessagef(err, "failed to execute env template %s", tpl.Name())
			log.WithError(err).Error("error")
			return nil, nil, err
		}
		// For env vars, we need to remove any new lines
		out = strings.ReplaceAll(out, "\n", "")
		cmdEnv = append(cmdEnv, fmt.Sprintf("%s=%s", key, out))
	}

	for cleanPath, realPath := range listener.tplTmpFileNames {
		cmdEnv = append(cmdEnv, fmt.Sprintf("GTE_FILES_%s=%s", cleanPath, realPath))
	}
	return &preparedExecutionResult{
		Command: cmdStr,
		Args:    cmdArgs,
		Env:     cmdEnv,
	}, nil, nil
}

func (listener *CompiledListener) HandleRequest(c *gin.Context, args map[string]interface{}, retryMap map[string]*HookShouldRetryInfo) (bool, *ListenerResponse, error) {
	if retryMap == nil {
		retryMap = make(map[string]*HookShouldRetryInfo)
	}

	// Keep track of what to store
	toStore := make(map[string]interface{})

	/*
		Create a new instance of the listener, to handle temporary files.

		On every new run, we store files in different temporary folders, and we need to populate
		the "files" map of the template with different values, which means pointing the "gte" function
		to a different listener!
	*/
	l, err := listener.clone()
	if err != nil {
		return false, nil, errors.WithMessage(err, "failed to clone listener")
	}

	timeStart := time.Now()

	out, errCommand := l.ExecCommand(args, toStore)
	defer l.cleanTemporaryFiles()

	for _, plugin := range l.plugins {
		if p, ok := plugin.(PluginHookPostExecute); ok {
			err := p.HookPostExecute(out)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errors.WithMessage(err, "failed to process post execution via plugin"))
				return true, nil, err
			}
		}
	}

	var retryDelay *time.Duration
	for _, plugin := range l.plugins {
		if p, ok := plugin.(PluginHookRetry); ok {
			id := p.Id()
			previousRetry := retryMap[id]

			var currentRetry *HookShouldRetryInfo
			if previousRetry == nil {
				currentRetry = &HookShouldRetryInfo{
					RetryCount: 1,
				}
			} else {
				currentRetry = previousRetry
				currentRetry.RetryCount++
			}

			currentRetry.Elapsed = time.Now().Sub(timeStart)

			retryMap[id] = currentRetry

			delayPtr, newArgs, err := p.HookShouldRetry(currentRetry, args, out)
			if err != nil {
				// If there is an error on retry, we should trigger the listener error handler
				errCommand = errors.WithMessage(err, "failed to perform retry")
				break
			}

			if delayPtr != nil {
				retryDelay = delayPtr
				args = newArgs
				break
			}
		}
	}

	if retryDelay != nil {
		// We should retry!
		l.log.Infof("retrying command in %s", retryDelay.String())
		time.Sleep(*retryDelay)
		return l.HandleRequest(c, args, retryMap)
	}

	if errCommand != nil {
		err := errors.WithMessagef(errCommand, "failed to execute listener %s", l.route)
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
	tplTmpFileNamesOriginalPaths := make(map[string]string)
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

	for key, filePath := range listener.tplTmpFileNamesOriginalPaths {
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
