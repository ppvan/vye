package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
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
	generateButton *ui.Button
	qrImageLabel   *ui.Static
	qrImage        *ui.Control
	copyButton     *ui.Button
	saveButton     *ui.Button

	textLabelData string
	qrImageData   []byte
}

func main() {
	runtime.LockOSThread() // Windows GUI must run on the OS thread

	win.CoInitializeEx(co.COINIT_APARTMENTTHREADED | co.COINIT_DISABLE_OLE1DDE)
	defer win.CoUninitialize()

	ShowMainWindow()
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

func (me *MyWindow) events() {
	me.qrImage.On().WmPaint(func() {
		var ps win.PAINTSTRUCT
		hdc, err := me.qrImage.Hwnd().BeginPaint(&ps)
		if err != nil {
			panic(err)
		}
		defer me.qrImage.Hwnd().EndPaint(&ps)

		backgroundBrush, _ := win.GetSysColorBrush(co.COLOR(co.COLOR_WINDOW)) // RGB red = 0x0000FF in BGR
		defer backgroundBrush.DeleteObject()
		hdc.FillRect(&ps.RcPaint, backgroundBrush)

		if me.qrImageData == nil {
			me.copyButton.Hwnd().EnableWindow(false)
			me.saveButton.Hwnd().EnableWindow(false)
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

		sz, _ := pic.Size()
		_, _ = pic.Render(hdc,
			win.POINT{},
			win.SIZE{Cx: ps.RcPaint.Right, Cy: ps.RcPaint.Bottom},
			win.POINT{X: 0, Y: sz.Cy},
			win.SIZE{Cx: sz.Cx, Cy: -sz.Cy},
		)
	})

	me.generateButton.On().BnClicked(func() {
		me.qrImageData = nil
		me.generateButton.SetText("Generating QR code...")
		me.generateButton.Hwnd().EnableWindow(false)
		me.qrImage.Hwnd().RedrawWindow(nil, 0, co.RDW_INVALIDATE)

		text := me.textEdit.Text()
		go func() {
			qrc, err := qrcode.New(text)
			if err != nil {
				fmt.Printf("could not generate QRCode: %v", err)
				return
			}

			var buf bytes.Buffer
			wr := nopCloser{Writer: &buf}

			w2 := standard.NewWithWriter(wr, standard.WithQRWidth(16))
			if err = qrc.Save(w2); err != nil {
				panic(err)
			}
			bitmapBuf, _ := pngToBitmapInMemory(buf.Bytes())

			me.wnd.UiThread(func() {
				me.qrImageData = bitmapBuf
				me.qrImage.Hwnd().RedrawWindow(nil, 0, co.RDW_INVALIDATE)
				me.generateButton.Hwnd().EnableWindow(true)
				me.generateButton.SetText("Generate QR code")
				me.copyButton.Hwnd().EnableWindow(true)
				me.saveButton.Hwnd().EnableWindow(true)
			})
		}()

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

	me.saveButton.On().BnClicked(func() {
		releaser := win.NewOleReleaser() // will release all COM objects created here
		defer releaser.Release()
		var fod *win.IFileSaveDialog
		_ = win.CoCreateInstance(
			releaser,
			&co.CLSID_FileSaveDialog,
			nil,
			co.CLSCTX_ALL,
			&fod,
		)

		fod.SetFileName("qr_code.png")
		fod.SetFileTypes([]win.COMDLG_FILTERSPEC{
			{Name: "PNG Image", Spec: "*.png"},
		})

		if ok, _ := fod.Show(me.wnd.Hwnd()); ok {
			item, _ := fod.GetResult(releaser)
			filePath, _ := item.GetDisplayName(co.SIGDN_FILESYSPATH)

			pngData, _ := bitmapToPngInMemory(me.qrImageData)

			os.WriteFile(filePath, pngData, 0644)
		}
	})
}

func pngToBitmapInMemory(pngData []byte) ([]byte, error) {
	pngReader := bytes.NewReader(pngData)

	img, _, err := image.Decode(pngReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PNG: %w", err)
	}

	var bmpBuffer bytes.Buffer

	// Note: The golang.org/x/image/bmp package only supports specific bit depths.
	err = bmp.Encode(&bmpBuffer, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode BMP: %w", err)
	}

	return bmpBuffer.Bytes(), nil
}

func bitmapToPngInMemory(bmpData []byte) ([]byte, error) {
	bmpReader := bytes.NewReader(bmpData)

	img, err := bmp.Decode(bmpReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode BMP: %w", err)
	}

	var pngBuffer bytes.Buffer

	err = png.Encode(&pngBuffer, img)
	if err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %w", err)
	}

	return pngBuffer.Bytes(), nil
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }
