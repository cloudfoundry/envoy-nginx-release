package main

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

	watcherErr := make(chan error)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					watcherErr <- errors.New("File watcher: unexpected event")
				}
				if event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Remove == fsnotify.Remove ||
					event.Op&fsnotify.Rename == fsnotify.Rename ||
					event.Op&fsnotify.Chmod == fsnotify.Chmod {

					/*
					* It is important to re-add because though the filepath hasn't changed,
					* it's a new file in the fs.
					* Maybe it watches the inode or something
					 */
					watcher.Add(filepath)

					err := callback()
					if err != nil {
						watcherErr <- err
					}
				}
			case err, ok := <-watcher.Errors:
				if err != nil {
					watcherErr <- err
				}
				if !ok {
					watcherErr <- errors.New("File watcher: unexpected error")
				}
			}
		}
	}()

	err = watcher.Add(filepath)
	if err != nil {
		return err
	}
	readyChan <- true

	return <-watcherErr
}
