package main

import (
	_ "embed"
	"runtime"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"
)

//go:embed gopher.bmp
var gopher []byte

type MyWindow struct {
	wnd *ui.Main
}

func main() {
	runtime.LockOSThread() // Windows GUI must run on the OS thread
	ShowMainWindow()
}

func ShowMainWindow() int {
	wnd := ui.NewMain(
		ui.OptsMain().
			Title("Red Window").
			Style(co.WS_CAPTION | co.WS_SYSMENU | co.WS_CLIPCHILDREN | co.WS_BORDER | co.WS_VISIBLE | co.WS_MINIMIZEBOX | co.WS_MAXIMIZE).
			Size(ui.Dpi(400, 300)),
	)

	me := &MyWindow{wnd}
	me.events()
	return wnd.RunAsMain()
}

func (me *MyWindow) events() {
	me.wnd.On().WmPaint(func() {
		var ps win.PAINTSTRUCT
		hdc, err := me.wnd.Hwnd().BeginPaint(&ps)
		if err != nil {
			panic(err)
		}
		defer me.wnd.Hwnd().EndPaint(&ps)

		rel := win.NewOleReleaser()
		defer rel.Release() // important: release your COM resources to avoid leaks

		stream, err := win.SHCreateMemStream(rel, gopher)
		if err != nil {
			panic(err.Error())
		}

		pic, err := win.OleLoadPicture(rel, stream, 0, true)
		if err != nil {
			panic(err.Error()) // a PNG file will crash here
		}

		redBrush, _ := win.GetSysColorBrush(co.COLOR(co.COLOR_GRAYTEXT)) // RGB red = 0x0000FF in BGR
		defer redBrush.DeleteObject()
		hdc.FillRect(&ps.RcPaint, redBrush)

		sz, _ := pic.Size()
		_, _ = pic.Render(hdc,
			win.POINT{},
			win.SIZE{Cx: ps.RcPaint.Right, Cy: ps.RcPaint.Bottom},
			win.POINT{X: 0, Y: sz.Cy},
			win.SIZE{Cx: sz.Cx, Cy: -sz.Cy},
		)
	})
}
