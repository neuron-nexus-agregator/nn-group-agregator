package app

import (
	"time"

	"agregator/group/internal/interfaces"
	"agregator/group/internal/service/groupmaker"
)

type App struct {
	maker   *groupmaker.GroupMaker
	timeOut time.Duration
	logger  interfaces.Logger
}

func New(diff, maxDistance, alpha float64, timeOut time.Duration, logger interfaces.Logger) *App {
	return &App{
		maker:   groupmaker.NewGroupMaker(diff, maxDistance, alpha, 1*time.Hour, logger),
		timeOut: timeOut,
		logger:  logger,
	}
}

func (a *App) Run() {
	for {
		err := a.maker.UpdateGroups()
		if err != nil {
			a.logger.Error(err.Error())
		}
		time.Sleep(a.timeOut)
	}
}
