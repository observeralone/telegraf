package global

import (
	"database/sql"
	"time"

	"github.com/influxdata/telegraf/plugins/easedbautil"
)

type Status struct {
	easedbautl.BaseStatus
}

func New(serverTag string, db *sql.DB) *Status {
	s := &Status{*easedbautl.NewBaseStatus(serverTag)}
	return s
}

func (g *Status) Fill(db *sql.DB) error {
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

	currTime := time.Now()

	rows, err := db.Query("SHOW global status")
	if err != nil {
		return err
	}
	defer rows.Close()

	values := make(map[string]interface{})
	for rows.Next() {
		var key string
		var val string

		if err = rows.Scan(&key, &val); err != nil {
			return err
		}

		values[key] = val
	}

	if g.CurrStatus != nil {
		g.LastStatus = g.CurrStatus
		g.LastTime = g.CurrTime
	}

	g.CurrStatus = values
	g.CurrTime = currTime

	ok = true
	return nil
}
