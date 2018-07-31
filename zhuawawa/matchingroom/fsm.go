//Package matchingroom 游戏匹配 状态机定义
package matchingroom

import (
	"fmt"
	"sync"
)

//FSMState 状态
type FSMState string

//FSMEvent 事件
type FSMEvent string

//FSMHandler 处理方法，并返回新的状态
type FSMHandler func() FSMState

//FSM 有限状态机
type FSM struct {
	mu       sync.Mutex                           // 排他锁
	state    FSMState                             // 当前状态
	handlers map[FSMState]map[FSMEvent]FSMHandler // 处理地图集，每一个状态都可以出发有限个事件，执行有限个处理
}

// 获取当前状态
func (f *FSM) getState() FSMState {
	return f.state
}

// 设置当前状态
func (f *FSM) setState(newState FSMState) {
	f.state = newState
}

//AddHandler 某状态添加事件处理方法
func (f *FSM) AddHandler(state FSMState, event FSMEvent, handler FSMHandler) *FSM {
	if _, ok := f.handlers[state]; !ok {
		f.handlers[state] = make(map[FSMEvent]FSMHandler)
	}
	if _, ok := f.handlers[state][event]; ok {
		fmt.Printf("[警告] 状态(%s)事件(%s)已定义过", state, event)
	}
	f.handlers[state][event] = handler
	return f
}

//Call 事件处理
func (f *FSM) Call(event FSMEvent) FSMState {
	defer f.mu.Unlock()
	f.mu.Lock()
	events := f.handlers[f.getState()]
	if events == nil {
		return f.getState()
	}
	if fn, ok := events[event]; ok {
		oldState := f.getState()
		f.setState(fn())
		newState := f.getState()
		fmt.Println("状态从 [", oldState, "] 变成 [", newState, "]")
	}
	return f.getState()
}

//NewFSM 实例化FSM
func NewFSM(initState FSMState) *FSM {
	return &FSM{
		state:    initState,
		handlers: make(map[FSMState]map[FSMEvent]FSMHandler),
	}
}
