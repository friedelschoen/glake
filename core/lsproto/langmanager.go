package lsproto

import (
	"context"
	"fmt"
	"sync"
)

type LangManager struct {
	Reg *Registration // accessed from editor

	man *Manager
	li  struct {
		sync.Mutex
		li     *LangInstance
		cancel context.CancelFunc
	}
}

func NewLangManager(man *Manager, reg *Registration) *LangManager {
	return &LangManager{Reg: reg, man: man}
}

func (lang *LangManager) instance(startCtx context.Context) (*LangInstance, error) {
	lang.li.Lock()
	defer lang.li.Unlock()

	// existing running instance
	if lang.li.li != nil {
		return lang.li.li, nil
	}

	// setup instance context
	ctx0 := context.Background() // TODO: editor ctx?
	ctx, cancel := context.WithCancel(ctx0)

	// call cancel if startCtx is done
	stop := context.AfterFunc(startCtx, cancel)
	defer stop()

	li, err := NewLangInstance(ctx, lang)
	if err != nil {
		cancel()
		err = lang.WrapError(err)
		return nil, err
	}
	lang.li.li = li
	lang.li.cancel = cancel

	// handle server/client abnormal early exit
	go func() {
		defer cancel()
		if err := li.Wait(); err != nil {
			lang.PrintWrapError(err)
		}
		// ensure this instance is cleared
		lang.li.Lock()
		defer lang.li.Unlock()
		if lang.li.li == li {
			lang.li.li = nil
		}
	}()

	return li, nil
}

// returns true if the instance was running
func (lang *LangManager) Close() (error, bool) {
	lang.li.Lock()
	defer lang.li.Unlock()
	if lang.li.li != nil {
		lang.li.cancel()
		lang.li.li = nil
		return nil, true
	}
	return nil, false
}

func (lang *LangManager) PrintWrapError(err error) {
	lang.man.Error(lang.WrapError(err))
}

func (lang *LangManager) WrapError(err error) error {
	return fmt.Errorf("lsproto(%s): %w", lang.Reg.Language, err)
}

func (lang *LangManager) WrapMsg(s string) string {
	return fmt.Sprintf("lsproto(%s): %v", lang.Reg.Language, s)
}
