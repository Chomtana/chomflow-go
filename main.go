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
						fmt.Println("test")
						fc.Next("cron5")
						return nil
					},
				}),
				"cron5": CronStateDefinition(&StateDefinitionCron{
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
						return nil
					},
				}),
			},
		},
	)

	select {}
}
