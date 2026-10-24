package main

import "github.com/gcash/bchd/txscript"

// stepper executes a script one instruction at a time. It keeps a copy of the
// engine after every instruction so the debugger can step backwards.
type stepper struct {
	vm *txscript.Engine

	// history holds the engine state after each instruction, starting with
	// the state before the first one.
	history []*txscript.Engine

	// done is set once execution has finished, successfully or not.
	done bool

	// err is why execution failed. It is nil while execution is running and
	// after it succeeds.
	err error
}

// newStepper returns a stepper that starts from the engine's current state.
func newStepper(vm *txscript.Engine) *stepper {
	return &stepper{
		vm:      vm,
		history: []*txscript.Engine{vm.Clone()},
	}
}

// next executes the next instruction. It does nothing once execution has
// finished, so the result stays put and the engine is never stepped past its
// end or an error.
func (s *stepper) next() {
	if s.done {
		return
	}
	done, err := s.vm.Step()
	switch {
	case err != nil:
		s.done = true
		s.err = err
	case done:
		s.done = true
		s.err = s.vm.CheckErrorCondition(true)
	default:
		s.history = append(s.history, s.vm.Clone())
	}
}

// back undoes the last instruction. Once execution has finished, it returns to
// the state before the final instruction and clears the result.
func (s *stepper) back() {
	switch {
	case s.done:
		// The final instruction is never added to the history, so the last
		// entry is the state just before it.
		s.done = false
		s.err = nil
	case len(s.history) > 1:
		s.history = s.history[:len(s.history)-1]
	default:
		return
	}
	s.vm = s.history[len(s.history)-1].Clone()
}
