package main

import (
	"errors"
	"fmt"
)

type Machine struct {
	Name string

	GlobalStorage *Storage

	Flows map[string]*Flow

	StateStorages    map[string]*Storage
	StateDefinitions map[string]*StateDefinition
}

type MachineDefinition struct {
	Name             string
	InitialState     string
	InitialFlowName  string
	StateDefinitions map[string]*StateDefinition
}

var machines map[string]*Machine

func (m *Machine) NewFlow(name string, state string) (*Flow, error) {
	existingFlow, hasExistingFlow := m.Flows[name]

	stateStorage, hasState := m.StateStorages[state]
	stateDefinition, hasStateDefinition := m.StateDefinitions[state]

	if !hasState || !hasStateDefinition {
		return nil, errors.New("State " + state + " not found")
	}

	if hasExistingFlow {
		if existingFlow.Context.IsRunning {
			return nil, errors.New("Flow " + name + " is running")
		}

		if localStorage, ok := existingFlow.LocalStorages[state]; ok {
			existingFlow.Context.LocalStorage = localStorage
		} else {
			localStorage = &Storage{}
			existingFlow.Context.LocalStorage = localStorage
			existingFlow.LocalStorages[state] = localStorage
		}

		existingFlow.Context.CurrentState = stateDefinition
		existingFlow.Context.StateStorage = stateStorage

		existingFlow.Context.ContextStorage.Clear()

		return existingFlow, nil
	} else {
		contextStorage := &Storage{}
		localStorage := &Storage{}
		flowStorage := &Storage{}

		ctx := &FlowContext{
			ContextStorage: contextStorage,
			LocalStorage:   localStorage,
			FlowStorage:    flowStorage,
			StateStorage:   stateStorage,
			GlobalStorage:  m.GlobalStorage,
			CurrentState:   stateDefinition,
		}

		flow := &Flow{
			Name:        name,
			Context:     ctx,
			Machine:     m,
			FlowStorage: flowStorage,

			LocalStorages:       make(map[string]*Storage),
			PendingEventTrigger: make(map[string][]interface{}),
		}

		flow.LocalStorages[state] = localStorage

		ctx.Flow = flow

		m.Flows[name] = flow

		return flow, nil
	}
}

func (m *Machine) NewChildFlow(parentContext *FlowContext, name string, state string) (*Flow, error) {
	flow, err := m.NewFlow(parentContext.Flow.Name+":"+name, state)

	if err != nil {
		return flow, err
	}

	flow.FlowStorage = parentContext.FlowStorage
	flow.LocalStorages = parentContext.Flow.LocalStorages
	flow.Context.FlowStorage = parentContext.FlowStorage
	flow.Context.ParentFlow = parentContext.Flow

	if localStorage, ok := flow.LocalStorages[state]; ok {
		flow.Context.LocalStorage = localStorage
	} else {
		localStorage = &Storage{}
		flow.Context.LocalStorage = localStorage
		flow.LocalStorages[state] = localStorage
	}

	return flow, nil
}

func NewMachine(definition *MachineDefinition) (*Machine, error) {
	globalStorage := &Storage{}
	_, hasInitialStateDefinition := definition.StateDefinitions[definition.InitialState]

	if !hasInitialStateDefinition {
		return nil, errors.New("Initial state " + definition.InitialState + " not found")
	}

	m := Machine{
		Name:             definition.Name,
		GlobalStorage:    globalStorage,
		Flows:            make(map[string]*Flow),
		StateDefinitions: make(map[string]*StateDefinition),
		StateStorages:    make(map[string]*Storage),
	}

	for key, value := range definition.StateDefinitions {
		m.StateDefinitions[key] = value
		m.StateStorages[key] = &Storage{}
	}

	flow, err := m.NewFlow(definition.InitialFlowName, definition.InitialState)

	if err != nil {
		return nil, err
	}

	m.Flows[definition.InitialFlowName] = flow

	fmt.Println("run1")

	// Let it run on its own
	go flow.Context.Run(nil)

	return &m, nil
}
