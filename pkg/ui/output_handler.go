package ui

import (
	"log"
	"sync"

	"github.com/example/grpc-plugin-app/pkg/plugin"
)

// outputHandler implements plugin.OutputHandler for the main application
type outputHandler struct {
	pluginName string
	mutex      sync.Mutex
}

func NewOutputHandler(pluginName string) plugin.OutputHandler {
	return &outputHandler{pluginName: pluginName}
}

func (h *outputHandler) OnOutput(msg string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	log.Printf("[%s] %s", h.pluginName, msg)
	return nil
}

func (h *outputHandler) OnProgress(p plugin.Progress) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	log.Printf("[%s] Progress: %.1f%% (%s - Step %d/%d)",
		h.pluginName, p.PercentComplete, p.Stage, p.CurrentStep, p.TotalSteps)
	return nil
}

func (h *outputHandler) OnError(code, message, details string) error {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if details != "" {
		log.Printf("[%s] Error %s: %s\nDetails: %s", h.pluginName, code, message, details)
	} else {
		log.Printf("[%s] Error %s: %s", h.pluginName, code, message)
	}
	return nil
}
