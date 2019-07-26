package fakes

import "fmt"

type Logger struct {
	PrintlnCall struct {
		Receives struct {
			Message []interface{}
		}
		Messages []string
	}
}

func (l *Logger) Println(v ...interface{}) {
	l.PrintlnCall.Receives.Message = v

	l.PrintlnCall.Messages = append(l.PrintlnCall.Messages, fmt.Sprintln(v...))
}
