package app

import (
	"fmt"
	"os"

	"code.cloudfoundry.org/envoy-nginx/parser"
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
	// TODO: We should not have to create this file.
	// hpcloud/tail will wait for the file to exist
	// so we can wait for nginx to creaet it.
	err := os.WriteFile(errorLog, []byte(""), parser.FilePerm)
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
