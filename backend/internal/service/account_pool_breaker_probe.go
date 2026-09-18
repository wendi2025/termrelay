package service

// abandonProbe releases a half-open probe reservation when the scheduler chose
// an account but no upstream attempt was actually started (for example because
// the concurrency slot raced full or a session quota rejected the candidate).
// It must not change failure counters or cooldown deadlines.
func (b *accountPoolCircuitBreaker) abandonProbe(accountID int64) {
	if b == nil || accountID <= 0 {
		return
	}
	value, ok := b.states.Load(accountID)
	if !ok {
		return
	}
	state, _ := value.(*accountPoolCircuitState)
	if state == nil {
		return
	}
	state.mu.Lock()
	state.halfOpenInFlight = false
	state.mu.Unlock()
}
