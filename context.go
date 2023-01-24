package main

import (
	"errors"
	"fmt"
	"os"
	"sync"
)

type EventFunction func(*FlowContext, ...interface{}) error

type FlowContext struct {
	ContextStorage *Storage
	LocalStorage   *Storage
	FlowStorage    *Storage
	StateStorage   *Storage
	GlobalStorage  *Storage

	Flow         *Flow
	ParentFlow   *Flow
	CurrentState *StateDefinition

	Events map[string]EventFunction

	ContextAction *ContextAction
	IsRunning     bool
	Comitting     bool
}

type FlowAndState struct {
	Flow  string
	State string
}

type ContextAction struct {
	Stop      bool
	Parent    string
	NextState string
	Children  []FlowAndState
	Start     []FlowAndState
}

// =========================================
// UTILITY
// =========================================

func (ctx *FlowContext) GetContextOf(flow string) *FlowContext {
	targetFlow, found := ctx.Flow.Machine.Flows[flow]

	if !found {
		return nil
	}

	return targetFlow.Context
}

func (ctx *FlowContext) GetOrCreateContextOf(flow string) (*FlowContext, error) {
	targetContext := ctx.GetContextOf(flow)

	if targetContext != nil {
		return targetContext, nil
	}

	flowObj, err := ctx.Flow.Machine.NewFlow(flow, ctx.CurrentState.Name)

	if err != nil {
		return nil, err
	}

	return flowObj.Context, nil
}

// =========================================
// EVENT
// =========================================

func (ctx *FlowContext) RegisterEvent(name string, fn EventFunction) error {
	if val, ok := ctx.Events[name]; ok {
		if val != nil {
			return errors.New("Event " + name + " is already registered")
		}
	}

	ctx.Events[name] = fn

	return nil
}

func (ctx *FlowContext) UnregisterEvent(name string) {
	delete(ctx.Events, name)
}

func (ctx *FlowContext) StartTransaction() error {
	if ctx.Comitting {
		return errors.New("Context is comitting")
	}

	if ctx.ContextAction != nil {
		return errors.New("Context transaction has already started")
	}

	if !ctx.IsRunning {
		return errors.New("Flow isn't running")
	}

	ctx.ContextAction = &ContextAction{}

	return nil
}

// =========================================
// TRANSACTION ACTION
// =========================================

func (ctx *FlowContext) Next(state string) error {
	if ctx.Comitting {
		return errors.New("Context is comitting")
	}

	if ctx.ContextAction == nil {
		return errors.New("Context transaction is not started")
	}

	if ctx.ContextAction.Stop {
		return errors.New("Flow already stopped")
	}

	ctx.ContextAction.NextState = state

	return nil
}

func (ctx *FlowContext) Start(flow string, state string) error {
	if ctx.Comitting {
		return errors.New("Context is comitting")
	}

	if ctx.ContextAction == nil {
		return errors.New("Context transaction is not started")
	}

	ctx.ContextAction.Start = append(ctx.ContextAction.Start, FlowAndState{
		Flow:  flow,
		State: state,
	})

	return nil
}

func (ctx *FlowContext) Stop() error {
	if ctx.Comitting {
		return errors.New("Context is comitting")
	}

	if ctx.ContextAction == nil {
		return errors.New("Context transaction is not started")
	}

	if ctx.ContextAction.NextState != "" || ctx.ContextAction.Parent != "" || len(ctx.ContextAction.Children) > 0 {
		return errors.New("Flow has next state")
	}

	ctx.ContextAction.Stop = true

	return nil
}

func (ctx *FlowContext) Child(flow string, state string) error {
	if ctx.Comitting {
		return errors.New("Context is comitting")
	}

	if ctx.ContextAction == nil {
		return errors.New("Context transaction is not started")
	}

	if !ctx.ContextAction.Stop {
		ctx.Stop()
	}

	ctx.ContextAction.Children = append(ctx.ContextAction.Children, FlowAndState{
		Flow:  flow,
		State: state,
	})

	return nil
}

func (ctx *FlowContext) Parent(state string) error {
	if ctx.Comitting {
		return errors.New("Context is comitting")
	}

	if ctx.ContextAction == nil {
		return errors.New("Context transaction is not started")
	}

	if ctx.ParentFlow == nil {
		return errors.New("This context is root")
	}

	if !ctx.ContextAction.Stop {
		ctx.Stop()
	}

	ctx.ContextAction.Parent = state

	return nil
}

// =========================================
// STATE TRANSFER
// =========================================

func (ctx *FlowContext) killEffect() {
	for i := 0; i <= ctx.CurrentState.RetryCount; i++ {
		if err := ctx.CurrentState.onExit(ctx); err != nil {
			if i == ctx.CurrentState.RetryCount {
				ctx.CurrentState.onError(ctx, "exit", err)
			}
		} else {
			break
		}
	}
}

func (ctx *FlowContext) Kill() {
	if ctx.Comitting {
		return
	}

	if !ctx.IsRunning {
		return
	}

	ctx.IsRunning = false

	go ctx.killEffect()
}

func (ctx *FlowContext) ForceKill() {
	ctx.Comitting = false
	ctx.ContextAction = nil
	ctx.Kill()
}

func (ctx *FlowContext) Run(wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}

	if ctx.IsRunning {
		return
	}

	ctx.IsRunning = true

	ctx.Transaction(func() error {
		for i := 0; i <= ctx.CurrentState.RetryCount; i++ {
			if err := ctx.CurrentState.onEnter(ctx); err != nil {
				if i == ctx.CurrentState.RetryCount {
					ctx.CurrentState.onError(ctx, "enter", err)
					return err
				}
			}

			if err := ctx.CurrentState.onReady(ctx); err != nil {
				if i == ctx.CurrentState.RetryCount {
					ctx.CurrentState.onError(ctx, "ready", err)
					return err
				}
			}

			return nil
		}

		return nil
	})
}

func (ctx *FlowContext) RunNextState(nextState string) {
	// Let it run on its own
	ctx.IsRunning = false

	ctx.killEffect()

	flow, err := ctx.Flow.Machine.NewFlow(ctx.Flow.Name, nextState)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Next flow error: %s\n", err)
		ctx.IsRunning = true

		ctx.Kill()
		return
	}

	go flow.Context.Run(nil)
}

func (ctx *FlowContext) ForkNewFlow(newFlow FlowAndState) {
	flow, err := ctx.Flow.Machine.NewFlow(newFlow.Flow, newFlow.State)

	if err != nil {
		fmt.Fprintf(os.Stderr, "New flow error: %s\n", err)
		return
	}

	// Let it run on its own
	go flow.Context.Run(nil)
}

func (ctx *FlowContext) ForkChildFlow(newFlow FlowAndState) {
	flow, err := ctx.Flow.Machine.NewChildFlow(ctx, newFlow.Flow, newFlow.State)

	if err != nil {
		fmt.Fprintf(os.Stderr, "New flow error: %s\n", err)
		return
	}

	// Let it run on its own
	go flow.Context.Run(nil)
}

func (ctx *FlowContext) ForkParent(newState string) {
	if !ctx.ParentFlow.Context.IsRunning {
		ctx.ParentFlow.Context.RunNextState(newState)
	}
}

// =========================================
// COMMITTING
// =========================================

func (ctx *FlowContext) ResetCommitting() {
	ctx.Comitting = false
}

func (ctx *FlowContext) Commit() error {
	if ctx.ContextAction == nil {
		return errors.New("Context transaction is not started")
	}

	ctx.Comitting = true
	defer ctx.ResetCommitting()

	if ctx.ContextAction.Stop {
		ctx.Kill()
	}

	for _, state := range ctx.ContextAction.Start {
		ctx.ForkNewFlow(state)
	}

	for _, state := range ctx.ContextAction.Children {
		ctx.ForkChildFlow(state)
	}

	if ctx.ContextAction.Parent != "" {
		ctx.ForkParent(ctx.ContextAction.Parent)
	}

	if ctx.ContextAction.NextState != "" {
		defer ctx.RunNextState(ctx.ContextAction.NextState)
	}

	ctx.ContextAction = nil

	return nil
}

func (ctx *FlowContext) Rollback() {
	if ctx.Comitting {
		return
	}

	ctx.ContextAction = nil
}

func (ctx *FlowContext) commitInternal() {
	err := ctx.Commit()
	if err != nil {
		ctx.Comitting = true
	}
}

func (ctx *FlowContext) Transaction(fn func() error) error {
	ctx.StartTransaction()

	err := fn()
	if err != nil {
		ctx.Rollback()
		return err
	}

	go ctx.commitInternal()

	return nil
}
