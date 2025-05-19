package app

import (
	"time"

	"agregator/group/internal/endpoint/app"
)

type App struct {
	endpoint *app.App
}

func New(diff, maxDistance, alpha float64, sleepTime time.Duration) *App {
	return &App{
		endpoint: app.New(diff, maxDistance, alpha, sleepTime),
	}
}

func (a *App) Run() {
	a.endpoint.Run()
}
