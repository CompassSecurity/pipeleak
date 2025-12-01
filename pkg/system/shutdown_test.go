package system

import (
	"testing"
)

func TestRegisterGracefulShutdownHandler(t *testing.T) {
	t.Run("handler registration", func(t *testing.T) {
		handlerCalled := false
		handler := func() {
			handlerCalled = true
		}

		RegisterGracefulShutdownHandler(handler)

		if handlerCalled {
			t.Error("Handler should not be called immediately")
		}
	})

	t.Run("multiple handlers can be registered", func(t *testing.T) {
		handler1 := func() {}
		handler2 := func() {}

		RegisterGracefulShutdownHandler(handler1)
		RegisterGracefulShutdownHandler(handler2)
	})
}
