package usecase

import "time"

var pipelineTimingStages = []string{
	"context_assemble",
	"plan",
	"candidate_generate",
	"evidence_resolve",
	"aggregate",
	"video_state_enrich",
	"rank",
	"select",
	"fill",
	"final_item_build",
}

type PipelineTimer struct {
	startedAt time.Time
	timings   map[string]int64
}

func NewPipelineTimer() *PipelineTimer {
	timings := make(map[string]int64, len(pipelineTimingStages)+1)
	for _, stage := range pipelineTimingStages {
		timings[stage] = 0
	}
	timings["total"] = 0
	return &PipelineTimer{
		startedAt: time.Now(),
		timings:   timings,
	}
}

func (t *PipelineTimer) Observe(stage string, fn func() error) error {
	startedAt := time.Now()
	err := fn()
	t.record(stage, time.Since(startedAt))
	return err
}

func (t *PipelineTimer) record(stage string, duration time.Duration) {
	if t == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	t.timings[stage] = duration.Milliseconds()
}

func (t *PipelineTimer) Snapshot() map[string]int64 {
	result := make(map[string]int64, len(pipelineTimingStages)+1)
	if t == nil {
		for _, stage := range pipelineTimingStages {
			result[stage] = 0
		}
		result["total"] = 0
		return result
	}
	for _, stage := range pipelineTimingStages {
		result[stage] = t.timings[stage]
	}
	total := time.Since(t.startedAt)
	if total < 0 {
		total = 0
	}
	result["total"] = total.Milliseconds()
	return result
}
