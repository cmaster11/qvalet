package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type CompiledListener struct {
	config *ListenerConfig
	log    logrus.FieldLogger

	path string

	tplCmd  *template.Template
	tplArgs []*template.Template
}

type GoToExec struct {
	config *Config
}

func (gte *GoToExec) mountRoutes(engine *gin.Engine) {
	for path, listenerConfig := range gte.config.Listeners {
		log := logrus.WithField("listener", path)

		tplCmd, err := template.New(path).Funcs(sprig.TxtFuncMap()).Parse(listenerConfig.Command)
		if err != nil {
			log.WithError(err).WithField("template", listenerConfig.Command).Fatal("failed to parse listener command template")
		}

		var tplArgs []*template.Template
		for idx, str := range listenerConfig.Args {
			tpl, err := template.New(fmt.Sprintf("%s-%d", path, idx)).Funcs(sprig.TxtFuncMap()).Parse(str)
			if err != nil {
				log.WithError(err).WithField("template", tpl).Fatal("failed to parse listener args template")
			}
			tplArgs = append(tplArgs, tpl)
		}

		listener := &CompiledListener{
			config:  listenerConfig,
			log:     log,
			path:    path,
			tplCmd:  tplCmd,
			tplArgs: tplArgs,
		}

		handler := gte.getGinListenerHandler(listener)

		if len(listenerConfig.Methods) == 0 {
			engine.GET(path, handler)
			engine.POST(path, handler)
		} else {
			for _, method := range listenerConfig.Methods {
				engine.Handle(method, path, handler)
			}
		}
	}
}

func (gte *GoToExec) getGinListenerHandler(listener *CompiledListener) gin.HandlerFunc {
	return func(c *gin.Context) {
		args := make(map[string]interface{})

		// Use path params, if any
		for _, param := range c.Params {
			args[param.Key] = param.Value
		}

		if c.Request.Method != http.MethodGet {
			b := binding.Default(c.Request.Method, c.ContentType())
			if b == binding.Form || b == binding.FormMultipart {
				queryMap := make(map[string][]string)
				if err := c.ShouldBindWith(&queryMap, b); err != nil {
					c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to parse request form body"))
					return
				}
				for key, vals := range queryMap {
					if len(vals) > 0 {
						args[key] = vals[len(vals)-1]
					} else {
						args[key] = true
					}

					args["_form_"+key] = vals
				}
			} else {
				if err := c.ShouldBindWith(&args, b); err != nil {
					c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to parse request body"))
					return
				}
			}
		}

		// Always bind query
		{
			queryMap := make(map[string][]string)
			if err := c.ShouldBindQuery(&queryMap); err != nil {
				c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to parse request query"))
				return
			}
			for key, vals := range queryMap {
				if len(vals) > 0 {
					args[key] = vals[len(vals)-1]
				} else {
					args[key] = true
				}

				args["_query_"+key] = vals
			}
		}

		out, err := gte.execCommand(listener, args)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.WithMessagef(err, "failed to execute listener %s", listener.path))
			return
		}

		c.JSON(http.StatusOK, map[string]interface{}{
			"output": out,
		})
	}
}

func (gte *GoToExec) execCommand(listener *CompiledListener, args map[string]interface{}) (string, error) {
	log := listener.log

	if listener.config.LogArgs {
		log = log.WithField("args", args)
	}

	var tplCmdOutput bytes.Buffer
	if err := listener.tplCmd.Execute(&tplCmdOutput, args); err != nil {
		err := errors.WithMessage(err, "failed to execute command template")
		log.WithError(err).Error("error")
		return "", err
	}

	cmdStr := tplCmdOutput.String()

	var cmdArgs []string
	for _, tpl := range listener.tplArgs {
		var tplArgOutput bytes.Buffer
		if err := tpl.Execute(&tplArgOutput, args); err != nil {
			err := errors.WithMessagef(err, "failed to execute args template %s", tpl.Name())
			log.WithError(err).Error("error")
			return "", err
		}
		arg := tplArgOutput.String()
		cmdArgs = append(cmdArgs, arg)
	}

	out, err := exec.Command(cmdStr, cmdArgs...).CombinedOutput()
	if err != nil {
		msg := "failed to execute command"

		if listener.config.ReturnOutput {
			msg += ": " + string(out)
		}

		err := errors.WithMessage(err, msg)

		log := log
		if listener.config.LogOutput {
			log = log.WithField("output", string(out))
		}

		log.WithError(err).Error("error")
		return "", err
	}

	if listener.config.LogOutput {
		log = log.WithField("output", string(out))
	}

	log.Info("command executed")

	if listener.config.ReturnOutput {
		return string(out), nil
	}

	return "success", nil
}
