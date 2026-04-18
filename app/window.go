package main

import (
	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
)

type MyWindow struct {
	wnd            *ui.Main
	textLabel      *ui.Static
	textEdit       *ui.Edit
	generateButton *ui.Button
	qrImageLabel   *ui.Static
	qrImage        *ui.Control
	copyButton     *ui.Button
	saveButton     *ui.Button

	textLabelData string
	qrImageData   []byte
}

func ShowMainWindow() int {
	wnd := ui.NewMain(
		ui.OptsMain().
			Title(title).
			Size(ui.Dpi(720, 480)).ClassIconId(42),
	)

	textLabel := ui.NewStatic(wnd, ui.OptsStatic().
		Text("Text").
		Position(ui.Dpi(20, 20)),
	)

	textEdit := ui.NewEdit(wnd, ui.OptsEdit().
		Position(ui.Dpi(20, 40)).
		Height(ui.DpiY(120)).
		Width(ui.DpiX(320)).CtrlStyle(co.ES_MULTILINE|co.ES_LEFT|co.ES_AUTOVSCROLL),
	)

	generateButton := ui.NewButton(wnd, ui.OptsButton().
		Position(ui.Dpi(20, 405)).
		Text("Generate QR code").
		Height(ui.DpiY(30)).
		Width(ui.DpiX(320)),
	)

	qrImageLabel := ui.NewStatic(wnd, ui.OptsStatic().
		Text("QR code").
		Position(ui.Dpi(380, 20)),
	)
	qrImage := ui.NewControl(wnd, ui.OptsControl().Position(ui.Dpi(380, 40)).Size(ui.Dpi(320, 320)))
	copyButton := ui.NewButton(wnd, ui.OptsButton().Position(ui.Dpi(380, 405)).Height(ui.DpiY(30)).Width(ui.DpiX(150)).Text("Copy to clipboard"))
	saveButton := ui.NewButton(wnd, ui.OptsButton().Position(ui.Dpi(550, 405)).Height(ui.DpiY(30)).Width(ui.DpiX(150)).Text("Save"))

	me := &MyWindow{
		wnd:            wnd,
		textLabel:      textLabel,
		textEdit:       textEdit,
		generateButton: generateButton,
		qrImageLabel:   qrImageLabel,
		qrImage:        qrImage,
		copyButton:     copyButton,
		saveButton:     saveButton,
	}
	me.events()
	return wnd.RunAsMain()
}
