package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
)

const maxRounds = 6
const maxShotsPerUnit = 2000

func makeEntities(u UnitType) []Entity {
	arr := make([]Entity, u.Count)
	for i := 0; i < u.Count; i++ {
		arr[i] = Entity{
			ID:      fmt.Sprintf("%s_%d", u.ID, i),
			TypeID:  u.ID,
			Atk:     u.Atk,
			Bou:     u.Bou,
			HullMax: u.Cqu,
			Hull:    u.Cqu,
			SP:      u.Bou,
		}
	}
	return arr
}

func countByType(fleet []Entity) map[string]int {
	m := make(map[string]int)
	for i := range fleet {
		if fleet[i].Hull > 0 {
			m[fleet[i].TypeID]++
		}
	}
	return m
}

func countAlive(fleet []Entity) int {
	n := 0
	for i := range fleet {
		if fleet[i].Hull > 0 {
			n++
		}
	}
	return n
}

func collectAlive(fleet []Entity, buf []int) []int {
	buf = buf[:0]
	for i := range fleet {
		if fleet[i].Hull > 0 {
			buf = append(buf, i)
		}
	}
	return buf
}

// fireOneShot — exact port of JS fireOneShot.
// aliveIdx is the list of target indices with hull > 0 (stable during fire phase).
func fireOneShot(rng *rand.Rand, unit *Entity, targets []Entity, aliveIdx []int, pending map[string]*pendingDmg) string {
	if len(aliveIdx) == 0 {
		return ""
	}
	ti := aliveIdx[rng.Intn(len(aliveIdx))]
	target := &targets[ti]

	// Bounce: atk <= bou * 0.01 — still return typeId for RF
	if unit.Atk <= target.Bou*0.01 {
		return target.TypeID
	}

	key := target.ID
	p := pending[key]
	if p == nil {
		p = &pendingDmg{}
		pending[key] = p
	}

	effShield := target.SP - p.shield
	if effShield < 0 {
		effShield = 0
	}
	if effShield > 0 {
		abs := math.Min(effShield, unit.Atk)
		p.shield += abs
		rem := unit.Atk - abs
		if rem > 0 {
			p.hull += rem
		}
	} else {
		p.hull += unit.Atk
	}
	return target.TypeID
}

// fireOneUnit — exact port of JS fireOneUnit (RF loop, max 2000 shots).
func fireOneUnit(rng *rand.Rand, unit *Entity, targets []Entity, aliveIdx []int, pending map[string]*pendingDmg) {
	rfMap := RF[unit.TypeID]
	keep := true
	for shots := 0; keep; {
		shots++
		if shots > maxShotsPerUnit {
			break
		}
		hitID := fireOneShot(rng, unit, targets, aliveIdx, pending)
		if hitID == "" {
			break
		}
		rf := 0
		if rfMap != nil {
			rf = rfMap[hitID]
		}
		if rf <= 1 {
			keep = false
			break
		}
		// Prob continue = 1 - 1/rf ; stop if random >= that (same as JS)
		if rng.Float64() >= (1 - 1/float64(rf)) {
			keep = false
		}
	}
}

type roundFireResult struct {
	sdDef, hdDef, sdAtk, hdAtk float64
	explosions, checks         int
}

// fireRoundSimultaneous — exact port of JS fireRoundSimultaneous.
// Alive sets are computed once per side: hull is not applied until after all shots
// (pendingDamage), so the live target pool is identical to re-filtering every shot.
func fireRoundSimultaneous(rng *rand.Rand, atk, def []Entity, aliveAtkBuf, aliveDefBuf *[]int) roundFireResult {
	pendingOnDef := make(map[string]*pendingDmg)
	pendingOnAtk := make(map[string]*pendingDmg)

	aliveDef := collectAlive(def, *aliveDefBuf)
	*aliveDefBuf = aliveDef
	aliveAtk := collectAlive(atk, *aliveAtkBuf)
	*aliveAtkBuf = aliveAtk

	for i := range atk {
		if atk[i].Hull > 0 {
			fireOneUnit(rng, &atk[i], def, aliveDef, pendingOnDef)
		}
	}
	for i := range def {
		if def[i].Hull > 0 {
			fireOneUnit(rng, &def[i], atk, aliveAtk, pendingOnAtk)
		}
	}

	var sdDef, hdDef, sdAtk, hdAtk float64

	for i := range def {
		u := &def[i]
		p := pendingOnDef[u.ID]
		if p == nil {
			continue
		}
		sd := math.Min(u.SP, p.shield)
		u.SP -= sd
		sdDef += sd
		hd := p.hull + (p.shield - sd)
		if hd > 0 {
			u.Hull -= hd
			hdDef += hd
			if u.Hull < 0 {
				u.Hull = 0
			}
		}
	}
	for i := range atk {
		u := &atk[i]
		p := pendingOnAtk[u.ID]
		if p == nil {
			continue
		}
		sd := math.Min(u.SP, p.shield)
		u.SP -= sd
		sdAtk += sd
		hd := p.hull + (p.shield - sd)
		if hd > 0 {
			u.Hull -= hd
			hdAtk += hd
			if u.Hull < 0 {
				u.Hull = 0
			}
		}
	}

	// Explosions + shield reset — once per unit at end of round (ATK then DEF, same as [...atk,...def])
	explosions, checks := 0, 0
	applyExplosions := func(fleet []Entity) {
		for i := range fleet {
			u := &fleet[i]
			if u.Hull > 0 && u.Hull < u.HullMax*0.7 {
				checks++
				if rng.Float64() < 1-(u.Hull/u.HullMax) {
					u.Hull = 0
					explosions++
					continue
				}
			}
			if u.Hull > 0 {
				u.SP = u.Bou
			}
		}
	}
	applyExplosions(atk)
	applyExplosions(def)

	return roundFireResult{
		sdDef: math.Round(sdDef), hdDef: math.Round(hdDef),
		sdAtk: math.Round(sdAtk), hdAtk: math.Round(hdAtk),
		explosions: explosions, checks: checks,
	}
}

func lostStr(before, after map[string]int) string {
	ids := make([]string, 0, len(before))
	for id := range before {
		ids = append(ids, id)
	}
	sort.Strings(ids) // stable display (JS Object.entries order is insertion-order)

	var out []string
	for _, id := range ids {
		l := before[id] - after[id]
		if l > 0 {
			out = append(out, fmt.Sprintf("%dx %s", l, strings.ReplaceAll(id, "_", " ")))
		}
	}
	if len(out) == 0 {
		return "Aucune perte"
	}
	return strings.Join(out, ", ")
}

func lostCount(before, after map[string]int) int {
	s := 0
	for id, n := range before {
		s += n - after[id]
	}
	return s
}

// Simulate runs one full combat (max 6 rounds). rng must be non-nil.
func Simulate(ab, db []UnitType, rng *rand.Rand) SimResult {
	atkTotal := 0
	defTotal := 0
	for _, u := range ab {
		atkTotal += u.OrigCount
	}
	for _, u := range db {
		defTotal += u.OrigCount
	}

	var atk, def []Entity
	for _, u := range ab {
		atk = append(atk, makeEntities(u)...)
	}
	for _, u := range db {
		def = append(def, makeEntities(u)...)
	}

	aliveAtkBuf := make([]int, 0, len(atk))
	aliveDefBuf := make([]int, 0, len(def))
	log := make([]RoundLog, 0, maxRounds)

	for r := 0; r < maxRounds; r++ {
		atkAliveStart := countAlive(atk)
		defAliveStart := countAlive(def)
		if atkAliveStart == 0 || defAliveStart == 0 {
			break
		}

		atkBefore := countByType(atk)
		defBefore := countByType(def)

		res := fireRoundSimultaneous(rng, atk, def, &aliveAtkBuf, &aliveDefBuf)

		atkAfter := countByType(atk)
		defAfter := countByType(def)
		ra := countAlive(atk)
		rd := countAlive(def)

		log = append(log, RoundLog{
			Round:          r + 1,
			AtkAlive:       ra,
			DefAlive:       rd,
			AtkTotal:       atkTotal,
			DefTotal:       defTotal,
			DmgShieldOnDef: res.sdDef,
			DmgHullOnDef:   res.hdDef,
			DefLostStr:     lostStr(defBefore, defAfter),
			DefLostCount:   lostCount(defBefore, defAfter),
			DmgShieldOnAtk: res.sdAtk,
			DmgHullOnAtk:   res.hdAtk,
			AtkLostStr:     lostStr(atkBefore, atkAfter),
			AtkLostCount:   lostCount(atkBefore, atkAfter),
			ExplTotal:      res.explosions,
			ChecksTotal:    res.checks,
		})
		if ra == 0 || rd == 0 {
			break
		}
	}

	al := countAlive(atk)
	dl := countAlive(def)
	winner := "draw"
	if al > 0 && dl == 0 {
		winner = "atk"
	} else if dl > 0 && al == 0 {
		winner = "def"
	}

	atkSurv := make(map[string]int)
	defSurv := make(map[string]int)
	for i := range atk {
		if atk[i].Hull > 0 {
			atkSurv[atk[i].TypeID]++
		}
	}
	for i := range def {
		if def[i].Hull > 0 {
			defSurv[def[i].TypeID]++
		}
	}

	defLossDetail := make(map[string]DefLossDetail, len(db))
	for _, u := range db {
		surv := defSurv[u.ID]
		destroyed := u.OrigCount - surv
		rebuilt := 0
		if DefIDs[u.ID] {
			for k := 0; k < destroyed; k++ {
				if rng.Float64() < 0.70 {
					rebuilt++
				}
			}
		}
		defLossDetail[u.ID] = DefLossDetail{
			Destroyed: destroyed,
			Rebuilt:   rebuilt,
			PermLost:  destroyed - rebuilt,
			Remaining: surv + rebuilt,
		}
	}

	return SimResult{
		Winner:        winner,
		Log:           log,
		AtkSurv:       atkSurv,
		DefSurv:       defSurv,
		DefLossDetail: defLossDetail,
	}
}
