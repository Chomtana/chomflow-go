package main

import (
	"fmt"
)

func main() {
	NewMachine(
		&MachineDefinition{
			Name:            "test",
			InitialFlowName: "main",
			InitialState:    "start",
			StateDefinitions: map[string]*StateDefinition{
				"start": BasicStateDefinition(&StateDefinition{
					onReady: func(fc *FlowContext) error {
						fmt.Println("Start state")
						fc.Next("cron3")
						return nil
					},
					onExit: func(fc *FlowContext) error {
						fmt.Println("Exit start state")
						return nil
					},
				}),
				"cron3": CronStateDefinition(&StateDefinitionCron{
					cronExpr: "* * * * * *",
					cronFn: func(fc *FlowContext) error {
						counter, ok := fc.ContextStorage.TemporaryStorage["counter"].(int)
						if ok {
							counter++
							fc.ContextStorage.TemporaryStorage["counter"] = counter
						} else {
							fc.ContextStorage.TemporaryStorage["counter"] = 1
							counter = 1
						}

						fmt.Printf("cron %d\n", counter)

						if counter >= 3 {
							fc.Next("start")
						}
						return nil
					},
					StateDefinition: StateDefinition{
						onReady: func(fc *FlowContext) error {
							fmt.Println("Enter cron3")
							return nil
						},
						onExit: func(fc *FlowContext) error {
							fmt.Println("Exit cron3")
							return nil
						},
					},
				}),
				"center": BasicStateDefinition(&StateDefinition{
					onReady: func(fc *FlowContext) error {
						return nil
					},
				}),
			},
		},
	)

	select {}
}
