package poolbench

import "testing"

func TestSharedPoolNeedsFewerHandshakes(t *testing.T) {
	cfg := Config{Requests: 600, Concurrency: 8, Workers: 16, Seed: 4}
	res := map[Mode]Result{}
	for _, m := range []Mode{NoReuse, PerWorker, Shared} {
		r, err := Run(m, cfg)
		if err != nil {
			t.Fatal(err)
		}
		res[m] = r
	}
	if res[NoReuse].Handshakes != int64(cfg.Requests) {
		t.Errorf("sin reutilizar: %d handshakes, quería %d", res[NoReuse].Handshakes, cfg.Requests)
	}
	if !(res[Shared].Handshakes < res[PerWorker].Handshakes && res[PerWorker].Handshakes < res[NoReuse].Handshakes) {
		t.Errorf("orden inesperado: compartido %d, por worker %d, sin reutilizar %d",
			res[Shared].Handshakes, res[PerWorker].Handshakes, res[NoReuse].Handshakes)
	}
}
