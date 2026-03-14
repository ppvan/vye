package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"runtime"

	go_qr "github.com/piglig/go-qr"
	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"
	"golang.org/x/image/bmp"
)

//go:embed gopher.bmp
var gopher []byte

const title = "QR generator"

// https://github.com/piglig/go-qr

type MyWindow struct {
	wnd            *ui.Main
	textLabel      *ui.Static
	textEdit       *ui.Edit
	sizeLabel      *ui.Static
	sizeOptions    *ui.ComboBox
	generateButton *ui.Button
	qrImage        *ui.Control
	copyButton     *ui.Button
	saveButton     *ui.Button

	textLabelData string
	qrImageData   []byte
}

func main() {
	runtime.LockOSThread() // Windows GUI must run on the OS thread
	ShowMainWindow()
}

func ShowMainWindow() int {

	wnd := ui.NewMain(
		ui.OptsMain().
			Title(title).
			Size(ui.Dpi(720, 480)),
	)

	textLabel := ui.NewStatic(wnd, ui.OptsStatic().
		Text("Text").
		Position(ui.Dpi(20, 20)),
	)

	textEdit := ui.NewEdit(wnd, ui.OptsEdit().
		Position(ui.Dpi(20, 40)).
		Width(ui.DpiX(320)),
	)

	sizeLabel := ui.NewStatic(wnd, ui.OptsStatic().
		Text("Size").
		Position(ui.Dpi(20, 215)),
	)

	sizeOptions := ui.NewComboBox(wnd, ui.OptsComboBox().
		Position(ui.Dpi(20, 235)).
		Width(ui.DpiX(320)).
		Texts("Auto").
		Select(0),
	)

	generateButton := ui.NewButton(wnd, ui.OptsButton().
		Position(ui.Dpi(20, 430)).
		Text("Generate").
		Height(ui.DpiY(30)).
		Width(ui.DpiX(320)),
	)

	qrImage := ui.NewControl(wnd, ui.OptsControl().Position(ui.Dpi(380, 20)).Size(ui.Dpi(320, 320)))

	me := &MyWindow{
		wnd:            wnd,
		textLabel:      textLabel,
		textEdit:       textEdit,
		sizeLabel:      sizeLabel,
		sizeOptions:    sizeOptions,
		generateButton: generateButton,
		qrImage:        qrImage,
	}
	me.events()
	return wnd.RunAsMain()
}

func (me *MyWindow) events() {
	me.qrImage.On().WmPaint(func() {
		if me.qrImageData == nil {
			return
		}

		var ps win.PAINTSTRUCT
		hdc, err := me.qrImage.Hwnd().BeginPaint(&ps)
		if err != nil {
			panic(err)
		}
		defer me.qrImage.Hwnd().EndPaint(&ps)

		rel := win.NewOleReleaser()
		defer rel.Release() // important: release your COM resources to avoid leaks

		stream, err := win.SHCreateMemStream(rel, me.qrImageData)
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

	me.generateButton.On().BnClicked(func() {
		text := me.textEdit.Text()
		errCorLvl := go_qr.Low
		qr, err := go_qr.EncodeText(text, errCorLvl)
		if err != nil {
			panic(err)
		}
		config := go_qr.NewQrCodeImgConfig(10, 4)

		var buf bytes.Buffer
		qr.WriteAsPNG(config, &buf)

		bitmapBuf, _ := pngToBitmapInMemory(buf.Bytes())

		me.qrImageData = bitmapBuf
	})
}

func pngToBitmapInMemory(pngData []byte) ([]byte, error) {
	// Use bytes.NewReader to create an io.Reader from the input PNG byte slice
	pngReader := bytes.NewReader(pngData)

	// 1. Decode the PNG image data into a generic image.Image interface
	img, _, err := image.Decode(pngReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	// Create a bytes.Buffer to store the encoded BMP data in memory
	var bmpBuffer bytes.Buffer

	// 2. Encode the image.Image into BMP format, writing to the buffer
	// Note: The golang.org/x/image/bmp package only supports specific bit depths.
	err = bmp.Encode(&bmpBuffer, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode BMP: %w", err)
	}

	// Return the in-memory BMP byte slice
	return bmpBuffer.Bytes(), nil
}
