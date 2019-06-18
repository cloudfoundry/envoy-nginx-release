package main

import (
	"errors"

	fsnotify "github.com/fsnotify/fsnotify"
)

func WatchFile(filepath string, callback func() error) error {
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
					* It is important to re-add because though the filepath has changed it's a new file
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

	return <-watcherErr
}
