package application

import (
	"context"
)

type App struct {
	Cfg *Config
	Out *Output
	Ctx context.Context
}

func NewApp(args []string) (app *App, mErr *MultiErr) {
	app = new(App)
	app.Ctx = context.Background()
	app.Out = NewOutput()
	app.Cfg, mErr = NewConfig(args, app.Out)

	return
}
