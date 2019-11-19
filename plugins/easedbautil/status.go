package easedbautl

import (
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Status interface {
	GetProperty(property string) (string, error)
	GetPropertyDelta(property string) (int64, error)
}

type BaseStatus struct {
	LastStatus map[string]interface{}
	CurrStatus map[string]interface{}
	CurrTime   time.Time
	LastTime   time.Time
	ServerTag  string
	Locker     *sync.Mutex
}

func NewBaseStatus(serverTag string) *BaseStatus {
	return &BaseStatus{
		LastStatus: nil,
		CurrStatus: nil,
		LastTime:   time.Now(),
		CurrTime:   time.Now(),
		ServerTag:  serverTag,
		Locker:     &sync.Mutex{},
	}
}

func (g *BaseStatus) GetProperty(property string) (string, error) {
	g.Locker.Lock()
	defer g.Locker.Unlock()

	if g.CurrStatus == nil {
		return "", fmt.Errorf("errror getting [%s] property: CurrStatus is nil", g.ServerTag)
	}

	val, ok := g.CurrStatus[property]
	if ! ok {
		return "", fmt.Errorf("errror getting [%s] property %s doesnot exist", g.ServerTag, property)
	}

	return string(val.(string)), nil
}

func (g *BaseStatus) GetPropertyDelta(property string) (int64, error) {
	g.Locker.Lock()
	defer g.Locker.Unlock()

	if g.LastStatus == nil {
		return 0, fmt.Errorf("error getting [%s] propery delta value, property: %s, no history data yet", g.ServerTag, property)
	}

	if g.CurrStatus == nil {
		return 0, fmt.Errorf("errror getting [%s] property: CurrStatus is nil", g.ServerTag)
	}

	lastVal, ok := g.LastStatus[property]
	if ! ok {
		return 0, fmt.Errorf("errror getting [%s] property delta, history property  %s doesnot exist", g.ServerTag, property)
	}

	currVal, ok := g.CurrStatus[property]
	if ! ok {
		return 0, fmt.Errorf("errror getting [%s] property delta, property  %s doesnot exist", g.ServerTag, property)
	}

	lastNum, err := strconv.ParseInt(lastVal.(string), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("errror getting [%s] property delta, property  %s is not a number: %s", g.ServerTag, property, err)
	}

	currNum, err := strconv.ParseInt(currVal.(string), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("errror getting [%s] property delta, property  %s is not a number: %s", g.ServerTag, property, err)
	}

	return currNum - lastNum, nil
}
