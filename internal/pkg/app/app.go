package app

import (
	"time"

	"agregator/group/internal/endpoint/app"
	"agregator/group/internal/interfaces"
	"agregator/group/internal/service/rtchecker"
)

type App struct {
	endpoint *app.App
}

func New(diff, maxDistance, alpha float64, sleepTime time.Duration, logger interfaces.Logger) *App {
	return &App{
		endpoint: app.New(diff, maxDistance, alpha, sleepTime, rtchecker.New(logger), logger),
	}
}

func (a *App) Run() {
	a.endpoint.Run()
}
