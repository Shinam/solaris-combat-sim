package engine

// Rapidfire table — exact copy of JS RF
var RF = map[string]map[string]int{
	"nova_scout": {
		"spectre_s": 5, "helios_r": 5,
	},
	"fer_de_lance": {
		"atlas_cargo": 3, "spectre_s": 5, "helios_r": 5,
	},
	"cyclone": {
		"nova_scout": 6, "spectre_s": 5, "helios_r": 5, "tourelle_c": 10,
	},
	"bastion": {
		"spectre_s": 5, "helios_r": 5,
	},
	"pyro": {
		"spectre_s": 5, "helios_r": 5, "tourelle_c": 20, "canon_imp": 20,
		"canon_fais": 10, "proj_ion": 10, "canon_m": 5, "canon_sol": 5,
	},
	"nemesis": {
		"atlas_cargo": 3, "atlas_cargo_xl": 3, "nova_scout": 4, "fer_de_lance": 4,
		"cyclone": 4, "bastion": 7, "spectre_s": 5, "helios_r": 5,
	},
	"fulgurant": {
		"spectre_s": 5, "helios_r": 5, "nemesis": 2, "canon_imp": 10,
	},
	"titan_astral": {
		"atlas_cargo": 250, "atlas_cargo_xl": 250, "nova_scout": 200, "fer_de_lance": 100,
		"cyclone": 33, "bastion": 30, "pyro": 25, "fulgurant": 5,
		"spectre_s": 1250, "helios_r": 1250, "barge_col": 250, "manta_recup": 250,
		"nemesis": 15, "tourelle_c": 200, "canon_imp": 200, "canon_fais": 100,
		"proj_ion": 100, "canon_m": 50,
	},
}

// Defense unit type IDs (70% rebuild chance)
var DefIDs = map[string]bool{
	"tourelle_c": true, "canon_imp": true, "canon_fais": true, "canon_m": true,
	"proj_ion": true, "canon_sol": true, "bouclier_p": true, "bouclier_g": true,
}
