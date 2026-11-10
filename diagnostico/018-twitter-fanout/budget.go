package fanout

// Budget es un presupuesto de reintentos al estilo del RetryBudget de Finagle: cada petición nueva deposita
// Ratio fichas y cada reintento gasta una, con un mínimo de MinRetries para no quedar en cero con poco tráfico.
// Así, los reintentos nunca superan ~Ratio del tráfico, por lento que esté el servicio.
type Budget struct {
	Ratio      float64
	MinRetries int
	balance    float64
	spentMin   int
}

// Deposit se llama una vez por petición nueva.
func (b *Budget) Deposit() { b.balance += b.Ratio }

// TryWithdraw dice si se puede reintentar y gasta la ficha.
func (b *Budget) TryWithdraw() bool {
	if b.spentMin < b.MinRetries {
		b.spentMin++
		return true
	}
	if b.balance >= 1 {
		b.balance--
		return true
	}
	return false
}

// Load cuenta cuántos intentos le llegan a un servicio que falla (timeout) por cada petición.
// maxRetries es lo que haría cada cliente por su cuenta; budget puede ser nil (sin presupuesto).
func Load(requests, maxRetries int, failRate float64, budget *Budget) (attempts int) {
	fails := func(i, try int) bool { // determinista: el servicio lento "falla" en una fracción fija de intentos
		x := uint64(i*7919+try*104729) % 1000
		return float64(x) < failRate*1000
	}
	for i := 0; i < requests; i++ {
		if budget != nil {
			budget.Deposit()
		}
		for try := 0; ; try++ {
			attempts++
			if !fails(i, try) || try == maxRetries {
				break
			}
			if budget != nil && !budget.TryWithdraw() {
				break
			}
		}
	}
	return attempts
}
