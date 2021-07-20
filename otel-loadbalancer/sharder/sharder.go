package sharder

import (
	"log"
	"sync"

	lbdiscovery "github.com/otel-loadbalancer/discovery"
)

/*
	Load balancer will serve on an HTTP server exposing /jobs/<job_id>/targets <- these are configured using least connection
	Load balancer will need information about the collectors in order to set the URLs
	Keep a Map of what each collector currently holds and update it based on new scrape target updates
*/

// Create a struct that holds collector - and jobs for that collector
// This struct will be parsed into endpoint with collector and jobs info

type collector struct {
	Name       string
	NumTargets int
}

// Sharder makes decisions to distribute work among
// a number of OpenTelemetry collectors based on the number of targets.
// Users need to call SetTargets when they have new targets in their
// clusters and call Reshard to process the new targets and reshard.
type Sharder struct {
	m     sync.Mutex
	cache displayCache

	// TODO: guard with mutex where needed.
	targetsWaiting map[string]lbdiscovery.TargetData // temp buffer to keep targets that are waiting to be processed

	targets    map[string]lbdiscovery.TargetData // all current targets used by Sharder to make decisions
	collectors map[string]*collector             // all current collectors

	nextCollector *collector
	targetItems   map[string]*targetItem // TODO: Merge this with targets, there should be one source of truth for all target state.
}

// findNextCollector finds the next collector with less number of targets.
func (sharder *Sharder) findNextCollector() {
	for _, v := range sharder.collectors {
		if v.NumTargets < sharder.nextCollector.NumTargets {
			sharder.nextCollector = v
		}
	}
}

// SetTargets accepts the a list of targets that will be used to make
// load balancing decisions. This method should be called when where are
// new targets discovered or existing targets are shutdown.
func (sharder *Sharder) SetTargets(targets []lbdiscovery.TargetData) {
	// TODO: Guard lb.targetsWaiting.
	// Dump old data
	for k := range sharder.targetsWaiting {
		delete(sharder.targetsWaiting, k)
	}
	// Set new data
	for _, i := range targets {
		sharder.targetsWaiting[i.JobName+i.Target] = i
	}
}

// SetCollectors sets the set of collectors with key=collectorName, value=Collector object.
func (sharder *Sharder) SetCollectors(collectors []string) {
	if len(collectors) == 0 {
		log.Fatal("no collector instances present")
	}
	sharder.m.Lock()
	for _, i := range collectors {
		collector := collector{Name: i, NumTargets: 0}
		sharder.collectors[i] = &collector
	}
	sharder.m.Unlock()
	sharder.nextCollector = sharder.collectors[collectors[0]]
}

// Reshard needs to be called to process the new target updates.
// Until Reshard is called, old targets will be served.
func (sharder *Sharder) Reshard() {
	// TODO: Reshard needs to be safe for concurrent access.
	// Guard the sharder fields with a mutex where needed.
	sharder.removeOutdatedTargets()
	sharder.processWaitingTargets()
	sharder.updateCache()
}

// removeOutdatedTargets removes targets that are no longer available.
func (sharder *Sharder) removeOutdatedTargets() {
	for k := range sharder.targets {
		if _, ok := sharder.targetsWaiting[k]; !ok {
			delete(sharder.targets, k)
			sharder.collectors[sharder.targetItems[k].Collector.Name].NumTargets--
			delete(sharder.targetItems, k)
		}
	}
}

// processWaitingTargets processes the newly set targets.
func (sharder *Sharder) processWaitingTargets() {
	for k, v := range sharder.targetsWaiting {
		if _, ok := sharder.targetItems[k]; !ok {
			sharder.findNextCollector()
			sharder.targets[k] = v
			targetItem := targetItem{
				JobName:   v.JobName,
				Link:      linkJSON{"/jobs/" + v.JobName + "/targets"},
				TargetURL: v.Target,
				Label:     v.Labels,
				Collector: sharder.nextCollector,
			}
			sharder.nextCollector.NumTargets++
			sharder.targetItems[v.JobName+v.Target] = &targetItem
		}
	}
}

func NewSharder() *Sharder {
	return &Sharder{
		targetsWaiting: make(map[string]lbdiscovery.TargetData),
		targets:        make(map[string]lbdiscovery.TargetData),
		collectors:     make(map[string]*collector),
		targetItems:    make(map[string]*targetItem),
	}
}
