package agent

type SilentUserProxy struct{}

func NewSilentUserProxy() *SilentUserProxy {
	return &SilentUserProxy{}
}

func (s *SilentUserProxy) Ask(question string) (string, error) {
	return "", nil
}

func (s *SilentUserProxy) Confirm(message string) (bool, error) {
	return true, nil
}

func (s *SilentUserProxy) Choose(prompt string, options []string) (string, error) {
	if len(options) > 0 {
		return options[0], nil
	}
	return "", nil
}
