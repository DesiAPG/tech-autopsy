package capacityguard

import "testing"

func demo(t *testing.T) Pool {
	t.Helper()
	p, err := Load("testdata/us-east-1.json")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// El caso: el input pretendía unos pocos servidores de facturación y seleccionó todo S3.
func TestInputMalEscritoSeRechaza(t *testing.T) {
	plan, err := Evaluate(demo(t), "s3-*")
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Selected) != 30 || plan.Approved() {
		t.Fatalf("esperaba 30 nodos y rechazo, got %d aprobado=%v", len(plan.Selected), plan.Approved())
	}
	for _, i := range plan.Impacts {
		if i.OK() {
			t.Errorf("%s debería quedar bajo su mínimo: %d → %d (mín %d)", i.Subsystem, i.Before, i.After, i.Min)
		}
	}
}

func TestPocosServidoresDeFacturacionPasa(t *testing.T) {
	plan, _ := Evaluate(demo(t), "s3-billing-0[1-3]")
	if len(plan.Selected) != 3 || !plan.Approved() {
		t.Fatalf("esperaba 3 nodos aprobados, got %d aprobado=%v", len(plan.Selected), plan.Approved())
	}
	if got := len(Apply(demo(t), plan).Nodes); got != 27 {
		t.Fatalf("quedan %d nodos, esperaba 27", got)
	}
}

// Aunque ningún subsistema baje del mínimo, un lote grande se frena: retirar despacio.
func TestLoteGrandeSeFrenaAunqueRespeteMinimos(t *testing.T) {
	p := demo(t)
	p.Subsystems["billing"] = Subsystem{Min: 0}
	plan, _ := Evaluate(p, "s3-billing-*")
	if plan.BatchOK() || plan.Approved() {
		t.Fatalf("12 nodos superan el lote máximo de %d", plan.MaxBatch)
	}
}

func TestPatronSinCoincidenciasNoSeAprueba(t *testing.T) {
	plan, _ := Evaluate(demo(t), "s3-billng-*")
	if plan.Approved() {
		t.Fatal("un patrón que no selecciona nada no debe aprobarse")
	}
}
