// Package main - backend entry point
package main

import (
	"github.com/neurochar/workflows/internal/app"
	"github.com/neurochar/workflows/internal/app/config"
	"github.com/neurochar/workflows/internal/app/fxboot"
	"go.uber.org/fx"
)

func main() {
	cfg := config.LoadConfig("configs/base.yml", "configs/base.local.yml")

	appOptions := fxboot.WorkflowsAppGetOptionsMap(app.IDWorkflows, cfg)

	app := fx.New(
		fxboot.OptionsMapToSlice(appOptions)...,
	)

	app.Run()
}
