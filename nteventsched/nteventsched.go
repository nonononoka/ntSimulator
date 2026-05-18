package nteventsched

import (
	"container/heap"
)

// priority queueに突っ込むnetwork eventの型
type Event struct {
	eventTime float64
	eventId   int
	args      []any
	callback  func(args ...any)
}

type PriorityQueue []*Event

// priorityQueueがheapをimplementするためのメソッドたち

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	return pq[i].eventTime < pq[j].eventTime
}

func (pq PriorityQueue) Swap(i, j int) { pq[i], pq[j] = pq[j], pq[i] }

func (pq *PriorityQueue) Push(x any) {
	*pq = append(*pq, x.(*Event)) // type assertion
}

func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[:n-1]
	return item
}

type NtEventSched struct {
	events      PriorityQueue
	CurrentTime float64
	eventId     int
	logEnabled  bool
	verbose     bool
	packetLogs  map[string]*packetLog
	*NetworkGraph
}

func (nes *NtEventSched) Run() {
	for nes.events.Len() > 0 {
		event := heap.Pop(&nes.events).(*Event)
		nes.CurrentTime = event.eventTime
		event.callback(event.args...)
	}
}

func (nes *NtEventSched) RunUntil(endTime float64) {
	for nes.events.Len() > 0 {
		event := heap.Pop(&nes.events).(*Event)
		nes.CurrentTime = event.eventTime
		if(nes.CurrentTime > endTime){
			return
		}
		event.callback(event.args...)
	}
}

func (nes *NtEventSched) ScheduleEvent(eventTime float64, callback func(args ...any), args ...any) {
	heap.Push(&nes.events, &Event{eventTime: eventTime, eventId: nes.eventId, callback: callback, args: args})
	nes.eventId += 1
}

// heap.Initは既存の要素をヒープ順に並べ直すためのもので，nilスライスならそのままで動く．
func NewNtEventSched(logEnabled bool, verbose bool) *NtEventSched {
	sched := &NtEventSched{
		NetworkGraph: newNetworkGraph(),
		logEnabled:   logEnabled,
		verbose:      verbose,
		packetLogs:   make(map[string]*packetLog),
	}
	return sched
}
