package configgate

import "testing"

var lim = Limits{MaxFeatures: 200, MaxGrowth: 1.5}

func TestRejectsDoubledFile(t *testing.T) {
	prev, err := Load("testdata/v41.json")
	if err != nil {
		t.Fatal(err)
	}
	next, err := Load("testdata/v42.json")
	if err != nil {
		t.Fatal(err)
	}
	cs := Validate(next, prev, lim)
	if Passed(cs) {
		t.Fatal("el archivo con columnas duplicadas no debería pasar")
	}
	for _, c := range cs {
		if (c.Rule == "duplicados" || c.Rule == "crecimiento") && c.OK {
			t.Errorf("la regla %q debería fallar: %s", c.Rule, c.Detail)
		}
	}
}

func TestAcceptsSameShape(t *testing.T) {
	prev, _ := Load("testdata/v41.json")
	if cs := Validate(prev, prev, lim); !Passed(cs) {
		t.Fatalf("la versión buena debería pasar: %+v", cs)
	}
}

func TestRejectsOverLimit(t *testing.T) {
	big := File{Version: 1}
	for i := range 201 {
		big.Features = append(big.Features, Feature{Name: "f" + string(rune('a'+i%26)) + string(rune('a'+i/26)), Type: "Float32"})
	}
	for _, c := range Validate(big, big, lim) {
		if c.Rule == "límite" && c.OK {
			t.Fatal("201 features no deberían pasar un límite de 200")
		}
	}
}
