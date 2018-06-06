package controller

import (
	"container/heap"
	"fmt"
	"sync"
	"time"
)

// EvalCache allows storing and retrieving EvalStates in a thread-safe way.
type EvalCache struct {
	states map[string]*EvalState
	lock   sync.RWMutex
}

func NewEvalCache() *EvalCache {
	return &EvalCache{
		states: map[string]*EvalState{},
	}
}

func (e *EvalCache) GetOrCreate(id string) *EvalState {
	s, ok := e.Get(id)
	if !ok {
		s = NewEvalState(id)
		e.Put(s)
	}
	return s
}

func (e *EvalCache) Get(id string) (*EvalState, bool) {
	e.lock.RLock()
	s, ok := e.states[id]
	e.lock.RUnlock()
	return s, ok
}

func (e *EvalCache) Put(state *EvalState) {
	e.lock.Lock()
	e.states[state.id] = state
	e.lock.Unlock()
}

func (e *EvalCache) Del(id string) {
	e.lock.Lock()
	delete(e.states, id)
	e.lock.Unlock()
}

func (e *EvalCache) List() map[string]*EvalState {
	results := map[string]*EvalState{}
	e.lock.RLock()
	for id, state := range e.states {
		results[id] = state
	}
	e.lock.RUnlock()
	return results
}

// EvalState is the state of a specific object that is evaluated in the controller.
//
// TODO add logger / or helper to log / context
// TODO add a time before next evaluation -> backoff
// TODO add current/in progress ignoreOk
type EvalState struct {
	// id is the identifier of the evaluation. For example the invocation.
	id string

	// EvalLog keep track of previous evaluations of this resource
	log EvalLog

	// evalLock allows gaining exclusive access to this evaluation
	evalLock chan struct{}

	// dataLock ensures thread-safe read and writes to this state. For example appending and reading logs.
	dataLock sync.RWMutex
}

func NewEvalState(id string) *EvalState {
	e := &EvalState{
		log:      EvalLog{},
		id:       id,
		evalLock: make(chan struct{}, 1),
	}
	e.Free()
	return e
}

// Lock returns the single-buffer lock channel. A consumer has obtained exclusive access to this evaluation if it
// receives the element from the channel. Compared to native locking, this allows consumers to have option to implement
// backup logic in case an evaluation is locked.
//
// Example: `<- es.Lock()`
func (e *EvalState) Lock() chan struct{} {
	return e.evalLock
}

// Free releases the obtained exclusive access to this evaluation. In case the evaluation is already free, this function
// is a nop.
func (e *EvalState) Free() {
	select {
	case e.evalLock <- struct{}{}:
	default:
		// was already unlocked
	}
}

func (e *EvalState) ID() string {
	return e.id
}

func (e *EvalState) Len() int {
	e.dataLock.RLock()
	defer e.dataLock.RUnlock()
	return len(e.log)
}

func (e *EvalState) Get(i int) (EvalRecord, bool) {
	e.dataLock.RLock()
	defer e.dataLock.RUnlock()
	if i >= len(e.log) {
		return EvalRecord{}, false
	}
	return e.log[i], true
}

func (e *EvalState) Last() (EvalRecord, bool) {
	e.dataLock.RLock()
	defer e.dataLock.RUnlock()
	return e.log.Last()
}

func (e *EvalState) First() (EvalRecord, bool) {
	e.dataLock.RLock()
	defer e.dataLock.RUnlock()
	return e.log.First()
}

func (e *EvalState) Logs() EvalLog {
	e.dataLock.RLock()
	defer e.dataLock.RUnlock()
	logs := make(EvalLog, len(e.log))
	copy(logs, e.log)
	return logs
}

func (e *EvalState) Record(record EvalRecord) {
	e.dataLock.Lock()
	e.log.Record(record)
	e.dataLock.Unlock()
}

// EvalRecord contains all metadata related to a single evaluation of a controller.
type EvalRecord struct {
	// Timestamp is the time at which the evaluation started. As an evaluation should not take any significant amount
	// of time the evaluation is assumed to have occurred at a point in time.
	Timestamp time.Time

	// Cause is the reason why this evaluation was triggered. For example: 'tick' or 'notification' (optional).
	Cause string

	// Action contains the action that the evaluation resulted in, if any.
	Action Action

	// Error contains the error that the evaluation resulted in, if any.
	Error error

	// RulePath contains all the rules that were evaluated in order to complete the evaluation.
	RulePath []string
}

func NewEvalRecord() EvalRecord {
	return EvalRecord{
		Timestamp: time.Now(),
	}
}

// EvalLog is a time-ordered log of evaluation records. Newer records are appended to the end of the log.
type EvalLog []EvalRecord

func (e EvalLog) Len() int {
	return len(e)
}

func (e EvalLog) Last() (EvalRecord, bool) {
	if e.Len() == 0 {
		return EvalRecord{}, false
	}
	return e[len(e)-1], true
}

func (e EvalLog) First() (EvalRecord, bool) {
	if e.Len() == 0 {
		return EvalRecord{}, false
	}
	return e[0], true
}

func (e *EvalLog) Record(record EvalRecord) {
	*e = append(*e, record)
}

type heapCmdType string

const (
	heapCmdPush     heapCmdType = "push"
	heapCmdPop      heapCmdType = "pop"
	heapCmdUpdate   heapCmdType = "update"
	heapCmdLength   heapCmdType = "len"
	heapCmdHalt     heapCmdType = "halt"
	DefaultPriority             = 0
)

type heapCmd struct {
	cmd    heapCmdType
	input  interface{}
	result chan<- interface{}
}

type ConcurrentEvalStateHeap struct {
	heap      *EvalStateHeap
	cmdChan   chan *heapCmd
	closeChan chan bool
	init      sync.Once
}

func NewConcurrentEvalStateHeap(unique bool) *ConcurrentEvalStateHeap {
	h := &ConcurrentEvalStateHeap{
		heap:      NewEvalStateHeap(unique),
		cmdChan:   make(chan *heapCmd, 50),
		closeChan: make(chan bool),
	}
	h.Init()
	heap.Init(h.heap)
	return h
}

func (h *ConcurrentEvalStateHeap) Init() {
	h.init.Do(func() {
		go func() {
			for {
				select {
				case cmd := <-h.cmdChan:
					switch cmd.cmd {
					case heapCmdLength:
						cmd.result <- h.heap.Len()
					case heapCmdPush:
						heap.Push(h.heap, cmd.input)
					case heapCmdPop:
						cmd.result <- heap.Pop(h.heap)
					case heapCmdUpdate:
						i := cmd.input.(*HeapItem)
						if i.index < 0 {
							cmd.result <- h.heap.Update(i.EvalState)
						} else {
							cmd.result <- h.heap.UpdatePriority(i.EvalState, i.Priority)
						}
					case heapCmdHalt:
						return
					}
				case <-h.closeChan:
					return
				}
			}
		}()
	})
}

func (h *ConcurrentEvalStateHeap) Len() int {
	result := make(chan interface{})
	h.cmdChan <- &heapCmd{
		cmd:    heapCmdLength,
		result: result,
	}
	return (<-result).(int)
}

func (h *ConcurrentEvalStateHeap) Update(s *EvalState) *HeapItem {
	result := make(chan interface{})
	h.cmdChan <- &heapCmd{
		cmd:    heapCmdUpdate,
		result: result,
		input: &HeapItem{
			EvalState: s,
			index:     -1, // abuse index as a signal to not update priority
		},
	}
	return (<-result).(*HeapItem)
}

func (h *ConcurrentEvalStateHeap) UpdatePriority(s *EvalState, priority int) *HeapItem {
	result := make(chan interface{})
	h.cmdChan <- &heapCmd{
		cmd:    heapCmdUpdate,
		result: result,
		input: &HeapItem{
			EvalState: s,
			Priority:  priority,
		},
	}
	return (<-result).(*HeapItem)
}

func (h *ConcurrentEvalStateHeap) Push(s *EvalState) {
	h.cmdChan <- &heapCmd{
		cmd:   heapCmdPush,
		input: s,
	}
}

func (h *ConcurrentEvalStateHeap) PushPriority(s *EvalState, priority int) {
	h.cmdChan <- &heapCmd{
		cmd: heapCmdPush,
		input: &HeapItem{
			EvalState: s,
			Priority:  priority,
		},
	}
}

func (h *ConcurrentEvalStateHeap) Pop() *EvalState {
	result := make(chan interface{})
	h.cmdChan <- &heapCmd{
		cmd:    heapCmdPop,
		result: result,
	}
	return (<-result).(*EvalState)
}

func (h *ConcurrentEvalStateHeap) Close() error {
	h.closeChan <- true
	close(h.closeChan)
	close(h.cmdChan)
	return nil
}

type HeapItem struct {
	*EvalState
	Priority int
	index    int
}

type EvalStateHeap struct {
	heap   []*HeapItem
	items  map[string]*HeapItem
	unique bool
}

func NewEvalStateHeap(unique bool) *EvalStateHeap {
	return &EvalStateHeap{
		items:  map[string]*HeapItem{},
		unique: unique,
	}
}
func (h EvalStateHeap) Len() int {
	return len(h.heap)
}

func (h EvalStateHeap) Less(i, j int) bool {
	it := h.heap[i]
	jt := h.heap[j]

	// Check priorities (descending)
	if it.Priority > jt.Priority {
		return true
	} else if it.Priority < jt.Priority {
		return false
	}

	// If priorities are equal, compare timestamp (ascending)
	return ignoreOk(it.Last()).Timestamp.Before(ignoreOk(jt.Last()).Timestamp)
}

func (h EvalStateHeap) Swap(i, j int) {
	h.heap[i], h.heap[j] = h.heap[j], h.heap[i]
	h.heap[i].index = i
	h.heap[j].index = j
}

// Use heap.Push
// The signature of push requests an interface{} to adhere to the sort.Interface interface, but will panic if a
// a type other than *EvalState is provided.
func (h *EvalStateHeap) Push(x interface{}) {
	switch t := x.(type) {
	case *EvalState:
		h.pushPriority(t, DefaultPriority)
	case *HeapItem:
		h.pushPriority(t.EvalState, t.Priority)
	default:
		panic(fmt.Sprintf("invalid entity submitted: %v", t))
	}
}

func (h *EvalStateHeap) pushPriority(state *EvalState, priority int) {
	if h.unique {
		if _, ok := h.items[state.id]; ok {
			h.UpdatePriority(state, priority)
			return
		}
	}
	el := &HeapItem{
		EvalState: state,
		Priority:  priority,
		index:     h.Len(),
	}
	h.heap = append(h.heap, el)
	h.items[state.id] = el
}

// Use heap.Pop
func (h *EvalStateHeap) Pop() interface{} {
	if h.Len() == 0 {
		return nil
	}
	popped := h.heap[h.Len()-1]
	delete(h.items, popped.id)
	h.heap = h.heap[:h.Len()-1]
	return popped.EvalState
}

func (h *EvalStateHeap) Update(es *EvalState) *HeapItem {
	if existing, ok := h.items[es.id]; ok {
		existing.EvalState = es
		heap.Fix(h, existing.index)
		return existing
	}
	return nil
}

func (h *EvalStateHeap) UpdatePriority(es *EvalState, priority int) *HeapItem {
	if existing, ok := h.items[es.id]; ok {
		existing.Priority = priority
		existing.EvalState = es
		heap.Fix(h, existing.index)
		return existing
	}
	return nil
}

func ignoreOk(r EvalRecord, _ bool) EvalRecord {
	return r
}
