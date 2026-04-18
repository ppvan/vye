package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"github.com/rodrigocfd/windigo/co"
	"github.com/rodrigocfd/windigo/win"
	"github.com/yeqown/go-qrcode/v2"
	"github.com/yeqown/go-qrcode/writer/standard"
)

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
