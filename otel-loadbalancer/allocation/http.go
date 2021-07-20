package allocation

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/common/model"
)

// TODO: Why do we have an _ in _link?
// TODO: Add missing JSON tags.
type TargetItem struct {
	JobName   string
	Link      linkJSON
	TargetURL string
	Label     model.LabelSet
	Collector *Collector
}

type linkJSON struct {
	Link string `json:"_link"`
}

type collectorJSON struct {
	Link string            `json:"_link"`
	Jobs []targetGroupJSON `json:"targets"`
}

type targetGroupJSON struct {
	Targets []string       `json:"targets"`
	Labels  model.LabelSet `json:"labels"`
}

// TODO: Consider removing cache, generate responses on the fly.
type displayCache struct {
	displayJobs          map[string](map[string][]targetGroupJSON)
	displayCollectorJson map[string](map[string]collectorJSON)
	displayJobMapping    map[string]linkJSON
	displayTargetMapping map[string][]targetGroupJSON
}

func (allocator *Allocator) JobHandler(w http.ResponseWriter, r *http.Request) {
	allocator.jsonHandler(w, r, allocator.cache.displayJobMapping)
}

func (allocator *Allocator) TargetsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()["collector_id"]
	params := mux.Vars(r)
	if len(q) == 0 {
		targets := allocator.cache.displayCollectorJson[params["job_id"]]
		allocator.jsonHandler(w, r, targets)
		return
	}
	data := allocator.cache.displayTargetMapping[params["job_id"]+q[0]]
	allocator.jsonHandler(w, r, data)
}

func (s *Allocator) jsonHandler(w http.ResponseWriter, r *http.Request, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (allocator *Allocator) generateCache() {
	var compareMap = make(map[string][]TargetItem) // CollectorName+jobName -> TargetItem
	for _, targetItem := range allocator.targetItems {
		compareMap[targetItem.Collector.Name+targetItem.JobName] = append(compareMap[targetItem.Collector.Name+targetItem.JobName], *targetItem)
	}
	allocator.cache = displayCache{displayJobs: make(map[string]map[string][]targetGroupJSON), displayCollectorJson: make(map[string](map[string]collectorJSON))}
	for _, v := range allocator.targetItems {
		allocator.cache.displayJobs[v.JobName] = make(map[string][]targetGroupJSON)
	}
	for _, v := range allocator.targetItems {
		var jobsArr []TargetItem
		jobsArr = append(jobsArr, compareMap[v.Collector.Name+v.JobName]...)

		var targetGroupList []targetGroupJSON
		targetItemSet := make(map[string][]TargetItem)
		for _, m := range jobsArr {
			targetItemSet[m.JobName+m.Label.String()] = append(targetItemSet[m.JobName+m.Label.String()], m)
		}
		labelSet := make(map[string]model.LabelSet)
		for _, targetItemList := range targetItemSet {
			var targetArr []string
			for _, targetItem := range targetItemList {
				labelSet[targetItem.TargetURL] = targetItem.Label
				targetArr = append(targetArr, targetItem.TargetURL)
			}
			targetGroupList = append(targetGroupList, targetGroupJSON{Targets: targetArr, Labels: labelSet[targetArr[0]]})

		}
		allocator.cache.displayJobs[v.JobName][v.Collector.Name] = targetGroupList
	}
}

// updateCache gets called whenever Reshard gets called
func (allocator *Allocator) updateCache() {
	allocator.generateCache() // Create cached structure
	// Create the display maps
	allocator.cache.displayTargetMapping = make(map[string][]targetGroupJSON)
	allocator.cache.displayJobMapping = make(map[string]linkJSON)
	for _, vv := range allocator.targetItems {
		allocator.cache.displayCollectorJson[vv.JobName] = make(map[string]collectorJSON)
	}
	for k, v := range allocator.cache.displayJobs {
		for kk, vv := range v {
			allocator.cache.displayCollectorJson[k][kk] = collectorJSON{Link: "/jobs/" + k + "/targets" + "?collector_id=" + kk, Jobs: vv}
		}
	}
	for _, targetItem := range allocator.targetItems {
		allocator.cache.displayJobMapping[targetItem.JobName] = linkJSON{targetItem.Link.Link}
	}

	for k, v := range allocator.cache.displayJobs {
		for kk, vv := range v {
			allocator.cache.displayTargetMapping[k+kk] = vv
		}
	}
}
