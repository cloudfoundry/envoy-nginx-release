package app

import (
	"io/ioutil"

	"github.com/hpcloud/tail"
)

type LogTailer struct {
	logger logger
}

func NewLogTailer(logger logger) LogTailer {
	return LogTailer{
		logger: logger,
	}
}

func (l LogTailer) Tail(errorLog string) {
	// TODO: TailFile will wait till the file is created by nginx.
	// Can we skip writing it ourselves?
	err := ioutil.WriteFile(errorLog, []byte(""), 0755)
	if err != nil {
		// TODO: Log error
		panic(err)
	}

	t, err := tail.TailFile(errorLog, tail.Config{
		Poll:   true,
		ReOpen: true,
		Follow: true,
	})
	if err != nil {
		// TODO: Log error
		panic(err)
	}

	for line := range t.Lines {
		l.logger.Println(line.Text)
	}
}
