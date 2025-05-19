package app

import (
	"log"
	"time"

	"agregator/group/internal/service/groupmaker"
)

type App struct {
	maker   *groupmaker.GroupMaker
	timeOut time.Duration
}

func New(diff, maxDistance, alpha float64, timeOut time.Duration) *App {
	return &App{
		maker:   groupmaker.NewGroupMaker(diff, maxDistance, alpha, 1*time.Hour),
		timeOut: timeOut,
	}
}

func (a *App) Run() {
	for {
		err := a.maker.UpdateGroups()
		if err != nil {
			log.Default().Println("Error updating groups:", err)
		}
		time.Sleep(a.timeOut)
	}
}
