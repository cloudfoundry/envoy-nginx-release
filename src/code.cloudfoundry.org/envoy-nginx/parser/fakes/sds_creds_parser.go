package fakes

type SdsCredParser struct {
	GetCertAndKeyCall struct {
		CallCount int
		Receives  struct {
			SdsFile string
		}
		Returns struct {
			Cert  string
			Key   string
			Error error
		}
	}
}

func (e SdsCredParser) GetCertAndKey(sdsFile string) (string, string, error) {
	e.GetCertAndKeyCall.CallCount++
	e.GetCertAndKeyCall.Receives.SdsFile = sdsFile

	return e.GetCertAndKeyCall.Returns.Cert, e.GetCertAndKeyCall.Returns.Key, e.GetCertAndKeyCall.Returns.Error
}
