package engine

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ProgressFunc is called periodically during a batch (optional).
type ProgressFunc func(done, total, aw, dw, dr int)

// RunBatch runs n independent simulations in parallel and aggregates
// results exactly like the JS Web Worker loop.
func RunBatch(ab, db []UnitType, n int, onProgress ProgressFunc) BatchResult {
	if n < 1 {
		n = 1
	}

	workers := runtime.GOMAXPROCS(0)
	if workers > n {
		workers = n
	}
	if workers < 1 {
		workers = 1
	}

	type partial struct {
		aw, dw, dr, totRounds int
		atkSurvAcc            map[string]int
		defSurvAcc            map[string]int
		atkLossDist           map[string][]int
		defLossDist           map[string][]DefLossDetail
		roundsDist            []int
		lastLog               []RoundLog
		lastIndex             int
	}

	jobs := make(chan int, n)
	var wg sync.WaitGroup
	results := make([]partial, workers)

	var doneCount int64
	var awA, dwA, drA int64
	progStep := n / 40
	if progStep < 1 {
		progStep = 1
	}

	seedBase := time.Now().UnixNano()

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(wid int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(seedBase + int64(wid)*1_000_003 + 17))
			p := &results[wid]
			p.atkSurvAcc = make(map[string]int, len(ab))
			p.defSurvAcc = make(map[string]int, len(db))
			p.atkLossDist = make(map[string][]int, len(ab))
			p.defLossDist = make(map[string][]DefLossDetail, len(db))
			for _, u := range ab {
				p.atkSurvAcc[u.ID] = 0
				p.atkLossDist[u.ID] = make([]int, 0, n/workers+1)
			}
			for _, u := range db {
				p.defSurvAcc[u.ID] = 0
				p.defLossDist[u.ID] = make([]DefLossDetail, 0, n/workers+1)
			}
			p.roundsDist = make([]int, 0, n/workers+1)
			p.lastIndex = -1

			for i := range jobs {
				res := Simulate(ab, db, rng)
				switch res.Winner {
				case "atk":
					p.aw++
				case "def":
					p.dw++
				default:
					p.dr++
				}
				p.totRounds += len(res.Log)
				p.roundsDist = append(p.roundsDist, len(res.Log))

				for _, u := range ab {
					s := res.AtkSurv[u.ID]
					p.atkSurvAcc[u.ID] += s
					p.atkLossDist[u.ID] = append(p.atkLossDist[u.ID], u.OrigCount-s)
				}
				for _, u := range db {
					d, ok := res.DefLossDetail[u.ID]
					if !ok {
						d = DefLossDetail{Remaining: u.OrigCount}
					}
					p.defSurvAcc[u.ID] += res.DefSurv[u.ID]
					p.defLossDist[u.ID] = append(p.defLossDist[u.ID], d)
				}

				if i > p.lastIndex {
					p.lastIndex = i
					p.lastLog = res.Log
				}

				d := int(atomic.AddInt64(&doneCount, 1))
				atomic.AddInt64(&awA, int64(boolToInt(res.Winner == "atk")))
				atomic.AddInt64(&dwA, int64(boolToInt(res.Winner == "def")))
				atomic.AddInt64(&drA, int64(boolToInt(res.Winner == "draw")))
				if onProgress != nil && (d%progStep == 0 || d == n) {
					onProgress(d, n, int(atomic.LoadInt64(&awA)), int(atomic.LoadInt64(&dwA)), int(atomic.LoadInt64(&drA)))
				}
			}
		}(w)
	}

	for i := 0; i < n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()

	out := BatchResult{
		Type:        "done",
		N:           n,
		AtkSurvAcc:  make(map[string]int, len(ab)),
		DefSurvAcc:  make(map[string]int, len(db)),
		AtkLossDist: make(map[string][]int, len(ab)),
		DefLossDist: make(map[string][]DefLossDetail, len(db)),
		RoundsDist:  make([]int, 0, n),
		AB:          ab,
		DB:          db,
	}
	for _, u := range ab {
		out.AtkSurvAcc[u.ID] = 0
		out.AtkLossDist[u.ID] = make([]int, 0, n)
	}
	for _, u := range db {
		out.DefSurvAcc[u.ID] = 0
		out.DefLossDist[u.ID] = make([]DefLossDetail, 0, n)
	}

	bestLast := -1
	for _, p := range results {
		out.AW += p.aw
		out.DW += p.dw
		out.DR += p.dr
		out.TotRounds += p.totRounds
		out.RoundsDist = append(out.RoundsDist, p.roundsDist...)
		for id, v := range p.atkSurvAcc {
			out.AtkSurvAcc[id] += v
		}
		for id, v := range p.defSurvAcc {
			out.DefSurvAcc[id] += v
		}
		for id, arr := range p.atkLossDist {
			out.AtkLossDist[id] = append(out.AtkLossDist[id], arr...)
		}
		for id, arr := range p.defLossDist {
			out.DefLossDist[id] = append(out.DefLossDist[id], arr...)
		}
		if p.lastIndex > bestLast {
			bestLast = p.lastIndex
			out.LastLog = p.lastLog
		}
	}
	return out
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
