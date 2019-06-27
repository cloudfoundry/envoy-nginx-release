package fakes

type Logger struct {
	PrintlnCall struct {
		Receives struct {
			Message string
		}
		Messages []string
	}
}

func (l *Logger) Println(message string) {
	l.PrintlnCall.Receives.Message = message

	l.PrintlnCall.Messages = append(l.PrintlnCall.Messages, message)
}
