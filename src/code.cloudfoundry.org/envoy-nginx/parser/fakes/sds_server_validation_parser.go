package fakes

type SdsServerValidationParser struct {
	GetCACertCall struct {
		CallCount int
		Receives  struct {
			SdsFile string
		}
		Returns struct {
			CA    string
			Error error
		}
	}
}

func (s SdsServerValidationParser) GetCACert(sdsFile string) (string, error) {
	s.GetCACertCall.CallCount++
	s.GetCACertCall.Receives.SdsFile = sdsFile

	return s.GetCACertCall.Returns.CA, s.GetCACertCall.Returns.Error
}
