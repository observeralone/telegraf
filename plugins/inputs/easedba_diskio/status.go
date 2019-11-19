package easedba_diskio

import (
	"time"

	"github.com/influxdata/telegraf/plugins/easedbautil"
)

type Status struct {
	easedbautl.BaseStatus
}

func New(device string) *Status {
	s := &Status{*easedbautl.NewBaseStatus(device)}
	return s
}

func (g *Status) Fill(diskioFields map[string]interface{}) error {
	g.Locker.Lock()
	defer g.Locker.Unlock()

	ok := false
	defer func() {
		if ! ok {
			//clean history data if current fetch failed
			// otherwise the delta is not expected since they will cross multi intervals
			g.LastStatus = nil
			g.CurrStatus = nil
		}
	}()


	g.LastStatus = g.CurrStatus
	g.LastTime = g.CurrTime


	g.CurrTime = time.Now()
	g.CurrStatus = diskioFields

	ok = true
	return nil
}
