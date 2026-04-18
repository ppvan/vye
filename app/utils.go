package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"

	"golang.org/x/image/bmp"
)

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
