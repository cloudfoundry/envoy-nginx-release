package app

import (
	"fmt"
	"io"
)

type Logger struct {
	writer io.Writer
}

func NewLogger(writer io.Writer) *Logger {
	return &Logger{
		writer: writer,
	}
}

func (l *Logger) Println(v ...interface{}) {
	fmt.Fprintln(l.writer, fmt.Sprintln(v...))
}
