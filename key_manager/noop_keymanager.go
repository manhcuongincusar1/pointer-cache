package keymanager

func NewNoopManager() KeyManager {
	return &noop{}
}

// Noop: No Operation
type noop struct {
}

func (p *noop) Add(key string) bool {
	return true
}
func (p *noop) Size() int {
	return 0
}
func (p *noop) Delete(key string) {} // Delete the key
func (p *noop) Peek() (string, error) {
	return "", nil
} // Take the first option
