package fakes

type Tailer struct {
	TailCall struct {
		CallCount int
		Receives  struct {
			Path string
		}
		Returns struct {
			Error error
		}
	}
}

func (t *Tailer) Tail(path string) error {
	t.TailCall.CallCount++
	t.TailCall.Receives.Path = path

	return t.TailCall.Returns.Error
}
