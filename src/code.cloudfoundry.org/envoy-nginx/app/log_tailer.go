package app

import (
	"fmt"
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

func (l LogTailer) Tail(errorLog string) error {
	err := ioutil.WriteFile(errorLog, []byte(""), 0755)
	if err != nil {
		return fmt.Errorf("write error.log: %s", err)
	}

	t, err := tail.TailFile(errorLog, tail.Config{
		Poll:   true,
		ReOpen: true,
		Follow: true,
		// MustExist: false,
	})
	if err != nil {
		return fmt.Errorf("tail file: %s", err)
	}

	go func() {
		for line := range t.Lines {
			l.logger.Println(line.Text)
		}
	}()

	// TODO: Handle EOF?
	return nil
}
