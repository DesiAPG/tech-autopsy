package qslog

import "testing"

func TestSequenceDetectsEveryGap(t *testing.T) {
	cfg := Config{Changes: 5000, Replicas: 3, Loss: 0.05, Batch: 50, Skew: 40, Seed: 5}
	log := Log(cfg)
	seq := Sequence(cfg, log)
	if !seq.Identical {
		t.Fatal("con secuencia, las réplicas deberían terminar iguales al líder")
	}
	if seq.Gaps == 0 || seq.Gaps != seq.Resent {
		t.Fatalf("huecos %d, reenviados %d", seq.Gaps, seq.Resent)
	}
	ts := Timestamps(cfg, log)
	if ts.Identical || ts.Lost == 0 {
		t.Fatalf("con timestamps se esperaban pérdidas silenciosas: %+v", ts)
	}
}

func TestNoLossNoGaps(t *testing.T) {
	cfg := Config{Changes: 1000, Replicas: 2, Loss: 0, Batch: 50, Skew: 0, Seed: 1}
	if r := Sequence(cfg, Log(cfg)); r.Gaps != 0 || !r.Identical {
		t.Fatalf("sin pérdidas no debería haber huecos: %+v", r)
	}
}
