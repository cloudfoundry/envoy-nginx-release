package fakes

type SdsCredParser struct {
	GetCertAndKeyCall struct {
		CallCount int
		Returns   struct {
			Cert  string
			Key   string
			Error error
		}
	}
}

func (e SdsCredParser) GetCertAndKey() (string, string, error) {
	e.GetCertAndKeyCall.CallCount++

	return e.GetCertAndKeyCall.Returns.Cert, e.GetCertAndKeyCall.Returns.Key, e.GetCertAndKeyCall.Returns.Error
}
