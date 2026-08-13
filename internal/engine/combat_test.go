package engine

import (
	"math/rand"
	"testing"
)

func TestSimulateBasic(t *testing.T) {
	ab := []UnitType{{
		ID: "nova_scout", Name: "Chasseur Nova", Count: 10, OrigCount: 10,
		Atk: 50, Bou: 10, Cqu: 400,
	}}
	db := []UnitType{{
		ID: "tourelle_c", Name: "Tourelle", Count: 5, OrigCount: 5,
		Atk: 80, Bou: 20, Cqu: 200,
	}}
	rng := rand.New(rand.NewSource(42))
	res := Simulate(ab, db, rng)
	if res.Winner != "atk" && res.Winner != "def" && res.Winner != "draw" {
		t.Fatalf("bad winner %q", res.Winner)
	}
	if len(res.Log) == 0 || len(res.Log) > 6 {
		t.Fatalf("bad log length %d", len(res.Log))
	}
}

func TestBounceStillAllowsRF(t *testing.T) {
	// manta atk=1 vs bouclier_g bou=10000 → bounce (1 <= 100)
	ab := []UnitType{{
		ID: "manta_recup", Name: "Moissonneur", Count: 1, OrigCount: 1,
		Atk: 1, Bou: 10, Cqu: 1600,
	}}
	db := []UnitType{{
		ID: "bouclier_g", Name: "Grand Bouclier", Count: 1, OrigCount: 1,
		Atk: 1, Bou: 10000, Cqu: 10000,
	}}
	rng := rand.New(rand.NewSource(1))
	res := Simulate(ab, db, rng)
	// Both should often survive (bounce); at least combat should complete
	if res.Winner == "" {
		t.Fatal("empty winner")
	}
}

func TestBatch(t *testing.T) {
	ab := []UnitType{{
		ID: "fer_de_lance", Name: "Fer", Count: 20, OrigCount: 20,
		Atk: 150, Bou: 25, Cqu: 1000,
	}}
	db := []UnitType{{
		ID: "cyclone", Name: "Cyclone", Count: 5, OrigCount: 5,
		Atk: 400, Bou: 50, Cqu: 2700,
	}}
	res := RunBatch(ab, db, 50, nil)
	if res.N != 50 {
		t.Fatalf("n=%d", res.N)
	}
	if res.AW+res.DW+res.DR != 50 {
		t.Fatalf("sum winners %d", res.AW+res.DW+res.DR)
	}
	if len(res.RoundsDist) != 50 {
		t.Fatalf("roundsDist %d", len(res.RoundsDist))
	}
}

func BenchmarkSimulate(b *testing.B) {
	ab := []UnitType{
		{ID: "nova_scout", Count: 100, OrigCount: 100, Atk: 50, Bou: 10, Cqu: 400},
		{ID: "fer_de_lance", Count: 50, OrigCount: 50, Atk: 150, Bou: 25, Cqu: 1000},
	}
	db := []UnitType{
		{ID: "tourelle_c", Count: 200, OrigCount: 200, Atk: 80, Bou: 20, Cqu: 200},
		{ID: "canon_imp", Count: 50, OrigCount: 50, Atk: 100, Bou: 25, Cqu: 250},
	}
	rng := rand.New(rand.NewSource(99))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Simulate(ab, db, rng)
	}
}
