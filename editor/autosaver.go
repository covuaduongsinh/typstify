package editor

import (
	"log"
	"sync"
	"sync/atomic"
	"time"
)

// An auto saver should be bound to a dedicated editor.
type AutoSaver struct {
	duration time.Duration
	timer    *time.Timer
	// saveFunc is invoked when internal timer fired or autoSaver is reset.
	// Caller should use a closure to capture outer lexical scoped variables.
	saveFunc func() error
	// updateCnt records how many saving request this autosaver has received since last successful saving.
	updateCnt    atomic.Uint32
	isRunning    atomic.Bool
	lastSaveTime time.Time
	stopChan     chan struct{}
	stopOnce     sync.Once
}

func NewAutoSaver(duration time.Duration, saveFunc func() error) *AutoSaver {
	if duration > time.Second*10 {
		duration = time.Second * 10
	}

	return &AutoSaver{
		duration:     duration,
		saveFunc:     saveFunc,
		lastSaveTime: time.Time{},
		stopChan:     make(chan struct{}, 1), // use a non-blocking channel
	}
}

func (as *AutoSaver) Update() {
	as.updateCnt.Add(1)
}

func (as *AutoSaver) HasPendingChanges() bool {
	return as.updateCnt.Load() > 0
}

func (as *AutoSaver) IsIdle() bool {
	return as.updateCnt.Load() == 0 && (!as.lastSaveTime.IsZero()) && time.Since(as.lastSaveTime) > 5*time.Minute
}

func (as *AutoSaver) IdleDuration() time.Duration {
	return time.Since(as.lastSaveTime)
}

func (as *AutoSaver) run() {
	if !as.isRunning.CompareAndSwap(false, true) {
		return
	}

	go func() {
		defer as.isRunning.Store(false)

		select {
		case <-as.timer.C:
			_ = as.doSave()

		case <-as.stopChan:
			log.Println("Stopping auto saver...")
			if !as.timer.Stop() {
				select {
				case <-as.timer.C:
				default:
				}
			}
			_ = as.doSave()
			return
		}
	}()
}

func (as *AutoSaver) doSave() error {
	//log.Println("updateCnt: ", as.updateCnt.Load())
	if as.updateCnt.Load() > 0 && as.saveFunc != nil {
		if err := as.saveFunc(); err == nil {
			as.updateCnt.Store(0)
			as.lastSaveTime = time.Now()
		} else {
			log.Printf("autosave error: %v", err)
			return err
		}
	}

	return nil
}

func (as *AutoSaver) SaveNow(cb func(), async bool) {
	call := func() {
		err := as.doSave()
		if err == nil && cb != nil {
			cb()
		}
	}

	if async {
		go call()
	} else {
		call()
	}
}

func (as *AutoSaver) Start() {
	if as.timer != nil {
		as.timer.Reset(as.duration)
	} else {
		as.timer = time.NewTimer(as.duration)
	}

	as.run()
}

func (as *AutoSaver) Stop() {
	as.stopOnce.Do(func() {
		close(as.stopChan)
	})
}

func (as *AutoSaver) IsRunning() bool {
	return as.isRunning.Load()
}
