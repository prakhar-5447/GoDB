package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// startServer starts the gRPC server as a subprocess.
func startServer() *exec.Cmd {
	log.Println("Starting gRPC server...")
	cmd := exec.Command("go", "run", "./cmd/server/main.go")

	// Redirect output to console.
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	return cmd
}

func main() {
	// Start the server for the first time.
	cmd := startServer()

	// Create a new file watcher.
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create file watcher: %v", err)
	}
	defer watcher.Close()

	// Watch directories recursively.
	dirsToWatch := []string{"./cmd", "./internal"}
	for _, dir := range dirsToWatch {
		filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("Error walking path %s: %v", path, err)
				return nil
			}
			if info.IsDir() {
				if err := watcher.Add(path); err != nil {
					log.Printf("Failed to watch directory %s: %v", path, err)
				}
			}
			return nil
		})
	}

	log.Println("Watching for file changes...")

	// Debounce mechanism.
	var restartTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			log.Printf("Change detected: %s", event)

			// Reset debounce timer.
			if restartTimer != nil {
				restartTimer.Stop()
			}

			restartTimer = time.AfterFunc(debounceDuration, func() {
				log.Println("Restarting gRPC server...")

				// Kill the old process if it's running.
				if cmd != nil && cmd.Process != nil {
					log.Println("Killing old server process...")
					if err := cmd.Process.Kill(); err != nil {
						log.Printf("Error killing server process: %v", err)
					}
					_ = cmd.Wait() // Ensure the process has fully exited.
				}

				// Wait a moment to allow the OS to release the port.
				time.Sleep(10 * time.Second)

				// Start a new server process.
				cmd = startServer()
			})

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}
