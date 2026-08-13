package engine

// UnitType is one fleet line as sent by the frontend (stats already tech-boosted).
type UnitType struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Count     int     `json:"count"`
	OrigCount int     `json:"origCount"`
	Atk       float64 `json:"atk"`
	Bou       float64 `json:"bou"`
	Cqu       float64 `json:"cqu"`
	Fer       float64 `json:"fer"`
	Sil       float64 `json:"sil"`
	Xen       float64 `json:"xen"`
	Cargo     float64 `json:"cargo"`
}

// Entity is one individual ship/defense unit in combat.
type Entity struct {
	ID      string
	TypeID  string
	Atk     float64
	Bou     float64
	HullMax float64
	Hull    float64
	SP      float64
}

type pendingDmg struct {
	shield float64
	hull   float64
}

// RoundLog matches the JS log entry fields exactly.
type RoundLog struct {
	Round           int     `json:"round"`
	AtkAlive        int     `json:"atkAlive"`
	DefAlive        int     `json:"defAlive"`
	AtkTotal        int     `json:"atkTotal"`
	DefTotal        int     `json:"defTotal"`
	DmgShieldOnDef  float64 `json:"dmgShieldOnDef"`
	DmgHullOnDef    float64 `json:"dmgHullOnDef"`
	DefLostStr      string  `json:"defLostStr"`
	DefLostCount    int     `json:"defLostCount"`
	DmgShieldOnAtk  float64 `json:"dmgShieldOnAtk"`
	DmgHullOnAtk    float64 `json:"dmgHullOnAtk"`
	AtkLostStr      string  `json:"atkLostStr"`
	AtkLostCount    int     `json:"atkLostCount"`
	ExplTotal       int     `json:"explTotal"`
	ChecksTotal     int     `json:"checksTotal"`
}

// DefLossDetail matches JS defLossDetail[id].
type DefLossDetail struct {
	Destroyed int `json:"destroyed"`
	Rebuilt   int `json:"rebuilt"`
	PermLost  int `json:"permLost"`
	Remaining int `json:"remaining"`
}

// SimResult is the result of one combat simulation.
type SimResult struct {
	Winner        string                   `json:"winner"`
	Log           []RoundLog               `json:"log"`
	AtkSurv       map[string]int           `json:"atkSurv"`
	DefSurv       map[string]int           `json:"defSurv"`
	DefLossDetail map[string]DefLossDetail `json:"defLossDetail"`
}

// BatchResult matches the Web Worker "done" payload.
type BatchResult struct {
	Type        string                     `json:"type"`
	AW          int                        `json:"aw"`
	DW          int                        `json:"dw"`
	DR          int                        `json:"dr"`
	N           int                        `json:"n"`
	TotRounds   int                        `json:"totRounds"`
	DurationMs  int64                      `json:"durationMs,omitempty"`
	AtkSurvAcc  map[string]int             `json:"atkSurvAcc"`
	DefSurvAcc  map[string]int             `json:"defSurvAcc"`
	LastLog     []RoundLog                 `json:"lastLog"`
	AtkLossDist map[string][]int           `json:"atkLossDist"`
	DefLossDist map[string][]DefLossDetail `json:"defLossDist"`
	RoundsDist  []int                      `json:"roundsDist"`
	AB          []UnitType                 `json:"ab"`
	DB          []UnitType                 `json:"db"`
}
