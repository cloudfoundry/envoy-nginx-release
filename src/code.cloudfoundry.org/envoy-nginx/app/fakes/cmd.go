package fakes

type Cmd struct {
	RunCall struct {
		CallCount int
		Receives  []RunCallReceive
		Returns   []RunCallReturn
	}
}

type RunCallReceive struct {
	Binary string
	Args   []string
}

type RunCallReturn struct {
	Error error
}

func (c *Cmd) Run(binary string, args ...string) error {
	c.RunCall.CallCount++
	c.RunCall.Receives = append(c.RunCall.Receives, RunCallReceive{Binary: binary, Args: args})

	if len(c.RunCall.Returns) < c.RunCall.CallCount {
		return nil
	}

	return c.RunCall.Returns[c.RunCall.CallCount-1].Error
}
