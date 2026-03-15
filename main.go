package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"io"
	"runtime"
	"time"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/ui"
	"github.com/rodrigocfd/windigo/win"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
	"golang.org/x/image/bmp"
)

const title = "QR generator"

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
	copyButton := ui.NewButton(wnd, ui.OptsButton().Position(ui.Dpi(380, 430)).Height(ui.DpiY(30)).Width(ui.DpiX(150)).Text("Copy to clipboard"))
	saveButton := ui.NewButton(wnd, ui.OptsButton().Position(ui.Dpi(550, 430)).Height(ui.DpiY(30)).Width(ui.DpiX(150)).Text("Save"))

	me := &MyWindow{
		wnd:            wnd,
		textLabel:      textLabel,
		textEdit:       textEdit,
		sizeLabel:      sizeLabel,
		sizeOptions:    sizeOptions,
		generateButton: generateButton,
		qrImage:        qrImage,
		copyButton:     copyButton,
		saveButton:     saveButton,
	}
	me.events()
	return wnd.RunAsMain()
}

func (me *MyWindow) events() {
	me.qrImage.On().WmPaint(func() {
		var ps win.PAINTSTRUCT
		hdc, err := me.qrImage.Hwnd().BeginPaint(&ps)
		if err != nil {
			panic(err)
		}
		defer me.qrImage.Hwnd().EndPaint(&ps)

		if me.qrImageData == nil {
			return
		}

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
		qrc, err := qrcode.New(text)
		if err != nil {
			fmt.Printf("could not generate QRCode: %v", err)
			return
		}

		var buf bytes.Buffer
		wr := nopCloser{Writer: &buf}

		w2 := standard.NewWithWriter(wr, standard.WithQRWidth(40))
		if err = qrc.Save(w2); err != nil {
			panic(err)
		}

		bitmapBuf, _ := pngToBitmapInMemory(buf.Bytes())
		me.qrImageData = bitmapBuf

		me.qrImage.Hwnd().RedrawWindow(nil, 0, co.RDW_INVALIDATE)
	})

	me.copyButton.On().BnClicked(func() {
		if me.qrImageData == nil {
			return
		}

		hClip, err := win.OpenClipboard(win.HWND(0))
		if err != nil {
			fmt.Print(err)
			return
		}
		defer hClip.CloseClipboard()

		if err = hClip.EmptyClipboard(); err != nil {
			fmt.Print(err)
			return
		}

		// BMP file has a 14-byte BITMAPFILEHEADER — CF_DIB needs it stripped
		const bmpFileHeaderSize = 14
		dibData := me.qrImageData[bmpFileHeaderSize:]

		err = hClip.SetClipboardData(co.CF_DIB, dibData)
		if err != nil {
			fmt.Print(err)
			return
		}

		me.copyButton.SetText("QR image copied!")
		go func() {
			time.Sleep(2 * time.Second)
			me.wnd.UiThread(func() {
				me.copyButton.SetText("Copy to clipboard")
			})
		}()
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

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
