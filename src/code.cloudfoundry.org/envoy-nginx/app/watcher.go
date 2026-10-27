package app

import (
	"errors"

	fsnotify "github.com/fsnotify/fsnotify"
)

/* readyChan tell when the watcher is ready */
func WatchFile(filepath string, readyChan chan bool, callback func() error) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	// Buffered so the reporting goroutine never blocks if WatchFile already returned.
	watcherErr := make(chan error, 1)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					watcherErr <- errors.New("File watcher: unexpected event")
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename {

					/*
					* It is important to re-add because though the filepath hasn't changed,
					* it's a new file in the fs.
					* Maybe it watches the inode or something
					 */
					err := watcher.Add(filepath)
					if err != nil {
						watcherErr <- err
						return
					}

					err = callback()
					if err != nil {
						watcherErr <- err
						return
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					watcherErr <- errors.New("File watcher: unexpected error")
					return
				}
				if err != nil {
					watcherErr <- err
					return
				}
			}
		}
	}()

	err = watcher.Add(filepath)
	if err != nil {
		return err
	}
	/*
	 * readyChan may be shared by several watchers and only read once
	 * (see envoy.go), so do not block forever on it: if the watcher fails
	 * before anyone reads readyChan, return the error instead.
	 */
	select {
	case readyChan <- true:
	case err := <-watcherErr:
		return err
	}

	return <-watcherErr
}
