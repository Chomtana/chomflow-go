package main

type Flow struct {
	Name    string
	Context *FlowContext
	Machine *Machine

	FlowStorage   *Storage
	LocalStorages map[string]*Storage

	PendingEventTrigger map[string][]interface{}
}

func (f *Flow) Stop() {
	f.Context.ForceKill()
}
