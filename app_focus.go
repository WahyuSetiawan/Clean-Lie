package main

import (
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// focusWindow menampilkan dan memunculkan window yang sudah berjalan. D-Bus
// tidak menyediakan "activate" lintas desktop, jadi show + unminimise +
// always-on-top sesaat adalah upaya terbaik yang portabel di X11/Wayland.
func (a *App) focusWindow() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowUnminimise(a.ctx)
	wruntime.WindowShow(a.ctx)
	wruntime.WindowSetAlwaysOnTop(a.ctx, true)
	time.AfterFunc(250*time.Millisecond, func() {
		if a.ctx != nil {
			wruntime.WindowSetAlwaysOnTop(a.ctx, false)
		}
	})
}
