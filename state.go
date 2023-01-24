package main

import (
	"fmt"
	"time"

	"github.com/go-co-op/gocron"
)

type StateDefinitionFunction func(*FlowContext) error
type StateDefinitionErrorFunction func(*FlowContext, string, error)

func DefaultStateDefinitionFunction(fn StateDefinitionFunction) StateDefinitionFunction {
	if fn == nil {
		return func(*FlowContext) error {
			return nil
		}
	}

	return fn
}

func DefaultStateDefinitionErrorFunction(fn StateDefinitionErrorFunction) StateDefinitionErrorFunction {
	if fn == nil {
		return func(_ *FlowContext, errType string, err error) {
			fmt.Println("Chomflow error", errType)
			fmt.Println(err)
		}
	}

	return fn
}

type StateDefinition struct {
	Name       string
	RetryCount int
	onEnter    StateDefinitionFunction
	onReady    StateDefinitionFunction
	onExit     StateDefinitionFunction
	onError    StateDefinitionErrorFunction
}

type StateDefinitionCron struct {
	StateDefinition
	cronExpr      string
	cronFn        StateDefinitionFunction
	singletonMode bool
}

func BasicStateDefinition(def *StateDefinition) *StateDefinition {
	return &StateDefinition{
		Name:       def.Name,
		RetryCount: def.RetryCount,
		onEnter:    DefaultStateDefinitionFunction(def.onEnter),
		onReady:    DefaultStateDefinitionFunction(def.onReady),
		onExit:     DefaultStateDefinitionFunction(def.onExit),
		onError:    DefaultStateDefinitionErrorFunction(def.onError),
	}
}

func CronStateDefinition(def *StateDefinitionCron) *StateDefinition {
	return &StateDefinition{
		Name:       def.Name,
		RetryCount: def.RetryCount,
		onEnter:    DefaultStateDefinitionFunction(def.onEnter),
		onReady: func(fc *FlowContext) error {
			if value, ok := fc.ContextStorage.TemporaryStorage["___scheduler"]; ok {
				s := value.(*gocron.Scheduler)
				s.Stop()
			}

			s := gocron.NewScheduler(time.UTC)

			if def.singletonMode {
				s = s.SingletonMode()
			}

			fc.ContextStorage.TemporaryStorage["___scheduler"] = s

			_, err := s.CronWithSeconds(def.cronExpr).Do(func() {
				if def.cronFn == nil {
					return
				}

				fc.Transaction(func() error {
					if cronFnErr := def.cronFn(fc); cronFnErr != nil {
						def.onError(fc, "cron", cronFnErr)
						return cronFnErr
					}

					return nil
				})
			})

			if err != nil {
				return err
			}

			if def.onReady != nil {
				if err := def.onReady(fc); err != nil {
					return err
				}
			}

			s.StartAsync()

			return nil
		},
		onExit: func(fc *FlowContext) error {
			if value, ok := fc.ContextStorage.TemporaryStorage["___scheduler"]; ok {
				s := value.(*gocron.Scheduler)
				s.Stop()
			}

			if def.onExit != nil {
				return def.onExit(fc)
			}

			return nil
		},
		onError: DefaultStateDefinitionErrorFunction(def.onError),
	}
}
