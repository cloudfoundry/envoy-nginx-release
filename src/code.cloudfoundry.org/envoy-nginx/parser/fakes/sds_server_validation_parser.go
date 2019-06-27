package fakes

type SdsServerValidationParser struct {
	GetCACertCall struct {
		CallCount int
		Returns   struct {
			CA    string
			Error error
		}
	}
}

func (s SdsServerValidationParser) GetCACert() (string, error) {
	s.GetCACertCall.CallCount++

	return s.GetCACertCall.Returns.CA, s.GetCACertCall.Returns.Error
}
