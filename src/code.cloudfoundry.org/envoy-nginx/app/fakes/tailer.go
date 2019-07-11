package fakes

type Tailer struct {
	TailCall struct {
		CallCount int
		Receives  struct {
			Path string
		}
	}
}

func (t *Tailer) Tail(path string) {
	t.TailCall.CallCount++
	t.TailCall.Receives.Path = path
}
