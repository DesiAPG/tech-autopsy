package contentgate

// IPC es la plantilla del caso: la definición declara 21 campos.
var IPC = Template{Name: "IPC", Fields: 21}

// Provided es lo que el código del sensor entregaba de verdad.
const Provided = 20

// rule arma una regla de 21 criterios: comodín en todos salvo los indicados.
func rule(id string, set map[int]string) Rule {
	c := make([]string, IPC.Fields)
	for i := range c {
		c[i] = Wildcard
	}
	for i, v := range set {
		c[i] = v
	}
	return Rule{ID: id, Criteria: c}
}

// RuleMarch es como las reglas de marzo y abril: el campo 21 con comodín.
var RuleMarch = rule("2024-03-05", map[int]string{0: `^\\\\\.\\pipe\\`, 3: `(?i)rundll32`})

// RuleJuly es la del 19 de julio: por primera vez un criterio real en el campo 21 (índice 20).
var RuleJuly = rule("2024-07-19", map[int]string{0: `^\\\\\.\\pipe\\`, 20: `.+`})

// SampleEvent son los 20 valores que el sensor arma cuando se crea un named pipe (ilustrativos).
func SampleEvent() Event {
	e := make(Event, Provided)
	e[0] = `\\.\pipe\demo`
	e[3] = "svchost.exe"
	for i := range e {
		if e[i] == "" {
			e[i] = "-"
		}
	}
	return e
}

// Rings es un despliegue escalonado de ejemplo.
var Rings = []Ring{{"canario", 10}, {"anillo 1", 1_000}, {"anillo 2", 100_000}, {"todos", 8_500_000}}
