package contentgate

import (
	"errors"
	"testing"
)

func TestReglaDeMarzoNoTocaElCampo21(t *testing.T) {
	if err := ValidateAgainstInputs(RuleMarch, Provided); err != nil {
		t.Fatal(err)
	}
	if _, err := Match(RuleMarch, SampleEvent()); err != nil {
		t.Fatal(err)
	}
	if crashes(RuleMarch, SampleEvent()) {
		t.Fatal("con comodín en el campo 21 no debería caer, igual que en marzo y abril")
	}
}

// El validador del incidente la aprueba; el corregido la frena.
func TestReglaDeJulio(t *testing.T) {
	if err := ValidateAgainstDefinition(IPC, RuleJuly); err != nil {
		t.Fatalf("el validador viejo la aprobó en el incidente: %v", err)
	}
	if err := ValidateAgainstInputs(RuleJuly, Provided); err == nil {
		t.Fatal("el validador nuevo debe frenarla")
	}
	if _, err := Match(RuleJuly, SampleEvent()); !errors.Is(err, ErrOutOfBounds) {
		t.Fatalf("el intérprete seguro debe dar error controlado, got %v", err)
	}
	if !crashes(RuleJuly, SampleEvent()) {
		t.Fatal("el intérprete inseguro debe caer (en el kernel: pantalla azul)")
	}
}

func TestLosAnillosFrenanEnElCanario(t *testing.T) {
	r := Deploy(RuleJuly, SampleEvent(), Rings)
	if r.FailedRing != "canario" || !r.RolledBack || r.Crashed != 10 || len(r.Reached) != 1 {
		t.Fatalf("esperaba rollback en el canario con 10 equipos, got %+v", r)
	}
	if ok := Deploy(RuleMarch, SampleEvent(), Rings); ok.FailedRing != "" || len(ok.Reached) != 4 {
		t.Fatalf("la regla buena debe llegar a todos, got %+v", ok)
	}
}
