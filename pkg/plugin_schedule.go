package pkg

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gotoexec/pkg/plugin_schedule"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/migrate"
)

var _ PluginInterface = (*PluginSchedule)(nil)
var _ PluginLifecycle = (*PluginSchedule)(nil)
var _ PluginConfigNeedsDb = (*PluginSchedule)(nil)
var _ PluginHookMountRoutes = (*PluginSchedule)(nil)
var _ PluginConfig = (*PluginScheduleConfig)(nil)

const pluginScheduleUrlParamTimeKey = "__gteScheduleTime"
const pluginSchedulePayloadTimeKey = "__gteScheduleTime"
const pluginScheduleScanIntervalMin = 100 * time.Millisecond

// @formatter:off
/// [config]
const pluginScheduleRouteDefault = "/schedule"
const pluginScheduleScanIntervalRestDefault = 10 * time.Second

type PluginScheduleConfig struct {
	// List of allowed authentication methods, defaults to the listener ones
	Auth []*AuthConfig `mapstructure:"auth" validate:"dive"`

	// Route to append, defaults to [pluginScheduleRouteDefault]
	Route *string `mapstructure:"route"`

	// How frequently should the plugin check for events to execute?
	// Defaults to [pluginScheduleScanIntervalRestDefault]
	ScanInterval *time.Duration `mapstructure:"scanInterval"`
}

/// [config]
// @formatter:on

func (c *PluginScheduleConfig) NewPlugin(listener *CompiledListener) (PluginInterface, error) {
	return &PluginSchedule{
		NewPluginBase("schedule"),
		listener,
		c,
		false,
	}, nil
}

func (c *PluginScheduleConfig) IsUnique() bool {
	return false
}

type PluginSchedule struct {
	PluginBase
	listener *CompiledListener
	config   *PluginScheduleConfig
	runLoop  bool
}

func (p *PluginSchedule) Migrations() *migrate.Migrations {
	return plugin_schedule.Migrations
}

func (p *PluginSchedule) Clone(_ *CompiledListener) (PluginInterface, error) {
	return p, nil
}

func (p *PluginSchedule) NeedsDb() bool {
	return true
}

var pluginScheduleRegexTime = regexp.MustCompile(`^\d+$`)

// Evaluates all possible types of "time"
func parseParamTime(val string, refTime *time.Time) (*time.Time, error) {
	val = strings.TrimSpace(val)

	// Is it a duration?
	if d, err := time.ParseDuration(val); err == nil {
		if refTime != nil {
			newTime := (*refTime).Add(d)
			return &newTime, nil
		}

		t := time.Now().Add(d)
		return &t, nil
	}

	// Is it a unix timestamp?
	if pluginScheduleRegexTime.MatchString(val) {
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err == nil {
			// https://stackoverflow.com/questions/23929145/how-to-test-if-a-given-time-stamp-is-in-seconds-or-milliseconds
			if len(val) >= 12 {
				// Ms!
				t := time.UnixMilli(parsed)
				return &t, nil
			}

			t := time.Unix(parsed, 0)
			return &t, nil
		}
	}

	return nil, errors.Errorf("invalid time value: %s", val)
}

type PluginScheduleResult struct {
	TaskId int64 `json:"taskId"`
}

func (p *PluginSchedule) scheduleTask(scheduleTime time.Time, args map[string]interface{}) (*PluginScheduleResult, error) {
	if p.listener.dbWrapper == nil {
		return nil, errors.New("database not initialized")
	}

	db := p.listener.dbWrapper.DB()

	task := &plugin_schedule.ScheduledTask{
		Id:         0, // Auto increment
		ExecuteAt:  scheduleTime,
		ListenerId: p.listener.id,
		Args:       args,
	}

	_, err := db.NewInsert().Model(task).Exec(context.Background())
	if err != nil {
		return nil, errors.WithMessage(err, "failed to insert scheduled task in database")
	}

	return &PluginScheduleResult{task.Id}, nil
}

func (p *PluginSchedule) loopIteration() (bool, error) {
	/*
		In each loop we want to look for db entries which match our listener
	*/
	listenerId := p.listener.id
	if listenerId == "" {
		// If something is wrong with the internal setup, we cannot proceed
		return false, errors.New("listener id not initialized")
	}

	if p.listener.dbWrapper == nil {
		return false, errors.New("database not initialized")
	}

	db := p.listener.dbWrapper.DB()

	rowFound := false
	var processingError error

	err := db.RunInTx(context.Background(), nil, func(ctx context.Context, tx bun.Tx) error {
		task := new(plugin_schedule.ScheduledTask)

		err := tx.NewSelect().
			Model(task).
			Where("listener_id = ?", listenerId).
			Where("execute_at < ?", time.Now()).
			Limit(1).
			Order("execute_at ASC").
			// Create a lock on the row
			For("UPDATE SKIP LOCKED").
			Scan(ctx)
		if err != nil {
			if err == sql.ErrNoRows {
				// All good, not found
				return nil
			}

			return errors.WithMessage(err, "failed to exec select statement")
		}

		if task.Id == 0 {
			// Not found
			return nil
		}

		defer func() {
			_, err := tx.NewDelete().Model(task).Where("id = ?", task.Id).Exec(context.Background())
			if err != nil {
				p.listener.Logger().WithError(err).Error("failed to delete scheduled task")
			}
		}()

		rowFound = true

		args := task.Args
		args[pluginSchedulePayloadTimeKey] = task.ExecuteAt

		/*
			Once we have the task, we just execute it
		*/
		w := httptest.NewRecorder()
		writeOnlyContext, _ := gin.CreateTestContext(w)
		_, _, err = p.listener.HandleRequest(writeOnlyContext, task.Args, nil)
		if err != nil {
			processingError = errors.WithMessage(err, "failed to handle delayed request")
			return nil
		}

		_, _ = ioutil.ReadAll(w.Body)

		return nil
	})
	if err != nil {
		return rowFound, errors.WithMessage(err, "failed to process next task (db error)")
	}
	if processingError != nil {
		return rowFound, errors.WithMessage(processingError, "failed to process next task")
	}

	return rowFound, nil
}

func (p *PluginSchedule) loop() {
	for p.runLoop {
		now := time.Now()

		rowFound, err := p.loopIteration()
		if err != nil {
			p.listener.Logger().WithError(err).Warn("plugin schedule iteration failed")
		}

		elapsed := time.Now().Sub(now)

		// How long should we wait before the next loop?
		// If we found data, process fast
		if rowFound {
			time.Sleep(pluginScheduleScanIntervalMin - elapsed)
			continue
		}

		// Otherwise, use the at-rest delay
		delay := pluginScheduleScanIntervalRestDefault
		if p.config.ScanInterval != nil {
			delay = *p.config.ScanInterval
		}
		time.Sleep(delay - elapsed)
	}
}

func (p *PluginSchedule) HookMountRoutes(engine *gin.Engine) {
	route := pluginScheduleRouteDefault
	if p.config.Route != nil {
		route = *p.config.Route
	}

	handler := func(c *gin.Context) {
		scheduleTimeString := c.Param(pluginScheduleUrlParamTimeKey)
		scheduleTime, err := parseParamTime(scheduleTimeString, nil)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, errors.WithMessage(err, "failed to parse schedule time"))
			return
		}

		authConfig := p.config.Auth
		// No, this is not a bug. If the plugin auth config is defined, even if it has
		// length 0, we want to keep it :)
		if authConfig == nil {
			authConfig = p.listener.config.Auth
		}

		handled, args := prepareListenerRequestHandling(c, authConfig)
		if handled {
			return
		}

		// Remove the param key, and add the parsed time one
		delete(args, pluginScheduleUrlParamTimeKey)

		result, err := p.scheduleTask(*scheduleTime, args)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, errors.WithMessage(err, "failed to scheduled task"))
			return
		}

		c.AbortWithStatusJSON(http.StatusOK, result)
	}

	mountRoutesByMethod(engine, p.listener.config.Methods, fmt.Sprintf("%s%s/:%s", p.listener.route, route, pluginScheduleUrlParamTimeKey), handler)
}

func (p *PluginSchedule) OnStart() error {
	p.runLoop = true
	go p.loop()
	return nil
}

func (p *PluginSchedule) OnStop() {
	p.runLoop = false
}
