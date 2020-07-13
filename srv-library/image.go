package srvlibrary

import (
	"bytes"
	"encoding/hex"
	"hash/fnv"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sangjihaksik/share"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fogleman/gg"
	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
	"github.com/nfnt/resize"
	"golang.org/x/image/font"
)

const (
	imageUpdatedStringY = 5

	seatUlY      = 10
	seatUlX      = 10
	seatUlSize   = 12
	seatUlMargin = 5

	seatSize               = 38
	seatNumberMarginBottom = 3

	outlineWidth = 6

	locationExit       = "exit"
	locationBackground = "background"

	colorBackground  = "#f5f8fe"
	colorSeatOutline = "#e8e8e8"
	colorDisabled    = "#808080"
	colorSeat        = "#f5f5f5"
	colorUsing       = "#9fd7e7"
)

var (
	imgExit = mustLoadLoage("img_exit.png")

	imgDisabled = resize.Resize(seatSize, 0, mustLoadLoage("explanatory_type_line2.png"), resize.Lanczos2)
	imgPillar   = resize.Resize(seatSize, 0, mustLoadLoage("img_pillar.png"), resize.Lanczos2)

	imgWheelChair = resize.Resize(seatUlSize, 0, mustLoadLoage("explanatory_type_line2.png"), resize.Lanczos2)
	imgNotebook   = resize.Resize(seatUlSize, 0, mustLoadLoage("explanatory_type_line2.png"), resize.Lanczos2)

	pngEncoder = png.Encoder{
		CompressionLevel: png.BestCompression,
		BufferPool:       new(pngBufferPool),
	}

	fontUpdated    = mustLoadFont("malgunbd-subset.ttf", 20)
	fontSeatNumber = mustLoadFont("malgun-subset.ttf", 13)
)

func mustLoadFont(path string, points float64) font.Face {
	path = filepath.Join(dirDrawing, path)
	f, err := gg.LoadFontFace(path, points)
	if err != nil {
		panic(err)
	}

	return f
}

// public/static 폴더에서 가져옴
func mustLoadLoage(path string) image.Image {
	path = filepath.Join(dirStatic, path)

	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	img, _, err := image.Decode(fs)
	if err != nil && err != io.EOF {
		panic(err)
	}

	return img
}
func resizeImage(img image.Image, w int, h int) image.Image {
	imgSize := img.Bounds().Size()

	imgNew := image.NewRGBA(image.Rect(0, 0, w, h))

	draw := gg.NewContextForRGBA(imgNew)
	draw.DrawImageAnchored(
		img,
		0,
		0,
		float64(w)/float64(imgSize.X),
		float64(h)/float64(imgSize.Y),
	)

	return imgNew
}

type imageData struct {
	location map[string]seatLocationData

	draw *gg.Context

	background image.Image

	pngLock       sync.RWMutex
	pngBody       []byte
	pngBodyBuffer bytes.Buffer
	pngBodyLength string
	pngETag       string
}

type seatLocationData struct {
	X            int             `json:"x"`
	Y            int             `json:"y"`
	BorderTop    seatBorderStyle `json:"border-top"`
	BorderLeft   seatBorderStyle `json:"border-left"`
	BorderRight  seatBorderStyle `json:"border-right"`
	BorderBottom seatBorderStyle `json:"border-bottom"`
}
type seatBorderStyle struct {
	Thickness float64 `json:"thickness"`
	Color     string  `json:"color"`
}

func newImageData(path string) (data *imageData) {
	var drawingData struct {
		Width      int                         `json:"width"`
		Height     int                         `json:"height"`
		Background string                      `json:"background"`
		Location   map[string]seatLocationData `json:"location"`
	}

	path = filepath.Join(dirDrawing, path)
	fs, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	err = jsoniter.NewDecoder(fs).Decode(&drawingData)
	if err != nil {
		panic(err)
	}

	if drawingData.Location == nil {
		return new(imageData)
	}

	data = &imageData{
		location:   drawingData.Location,
		background: mustLoadLoage(drawingData.Background),
		draw:       gg.NewContext(drawingData.Width, drawingData.Height),
	}

	return data
}

func (d *imageData) Serve(w http.ResponseWriter, r *http.Request) {
	d.pngLock.RLock()
	defer d.pngLock.RUnlock()

	if d.pngBody == nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		if lastModified := r.Header.Get("If-None-Match"); len(lastModified) > 0 {
			if lastModified == d.pngETag {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		header := w.Header()
		header.Set("Content-Type", "image/png")
		header.Set("Content-Length", d.pngBodyLength)
		header.Set("ETag", d.pngETag)
		header.Set("Cache-Control", "max-age=60")

		w.WriteHeader(http.StatusOK)
		w.Write(d.pngBody)
	}
}

func (d *imageData) DrawImage(now time.Time, seatMap map[int]SeatState) {
	if seatMap == nil || d.location == nil {
		d.pngLock.Lock()
		defer d.pngLock.Unlock()

		d.pngBody = nil

		return
	}

	// 내용물 싹 지우기
	d.draw.SetHexColor(colorBackground)
	d.draw.Clear()

	// 배경 그리기
	sd, ok := d.location[locationBackground]
	if ok {
		d.draw.DrawImage(d.background, sd.X, sd.Y)
	}

	// exit
	sd, ok = d.location[locationExit]
	if ok {
		d.draw.DrawImage(imgExit, sd.X, sd.Y)
	}

	// pillar
	for k, sd := range d.location {
		if !strings.HasPrefix(k, "pillar") {
			continue
		}

		d.draw.DrawImage(imgPillar, sd.X, sd.Y)
	}

	drawBorder := func(sd seatBorderStyle, x1, y1, x2, y2 int) {
		if sd.Thickness == 0 {
			return
		}

		d.draw.SetLineWidth(sd.Thickness)
		d.draw.SetHexColor(sd.Color)

		d.draw.DrawLine(float64(x1), float64(y1), float64(x2), float64(y2))
		d.draw.Stroke()
	}

	for _, seat := range seatMap {
		sd, ok := d.location[seat.SeatNum]
		if !ok {
			continue
		}

		d.draw.SetHexColor(colorSeatOutline)
		d.draw.DrawRectangle(
			float64(sd.X)-outlineWidth,
			float64(sd.Y)-outlineWidth,
			seatSize+outlineWidth*2,
			seatSize+outlineWidth*2,
		)
		d.draw.Fill()
	}

	d.draw.SetFontFace(fontSeatNumber)
	for _, seat := range seatMap {
		sd, ok := d.location[seat.SeatNum]
		if !ok {
			continue
		}

		// 좌석 네모 그리기
		if seat.Disabled {
			d.draw.SetHexColor(colorDisabled)
		} else if seat.Using {
			d.draw.SetHexColor(colorUsing)
		} else {
			d.draw.SetHexColor(colorSeat)
		}
		d.draw.DrawRectangle(float64(sd.X), float64(sd.Y), seatSize, seatSize)
		d.draw.Fill()

		// 왼쪽 상단에 휠체어 노트북 그리기
		ulX := sd.X + seatUlX
		if seat.WheelChair {
			d.draw.DrawImage(imgWheelChair, ulX, sd.Y+seatUlY)
			ulX += seatUlSize + seatUlMargin
		}
		if seat.NoteBook {
			d.draw.DrawImage(imgNotebook, ulX, sd.Y+seatUlY)
		}

		d.draw.SetColor(color.Black)
		d.draw.DrawStringAnchored(
			strings.TrimLeft(seat.SeatNum, "0"),
			float64(sd.X)+float64(seatSize)/2,
			float64(sd.Y+seatSize-seatNumberMarginBottom),
			0.5,
			0,
		)

		if seat.Disabled {
			d.draw.DrawImage(imgDisabled, sd.X, sd.Y)
		}
	}

	for _, seat := range seatMap {
		sd, ok := d.location[seat.SeatNum]
		if !ok {
			continue
		}

		drawBorder(
			sd.BorderTop,
			sd.X, sd.Y-1,
			sd.X+seatSize, sd.Y-1,
		)
		drawBorder(
			sd.BorderBottom,
			sd.X, sd.Y+seatSize+1,
			sd.X+seatSize, sd.Y+seatSize+1,
		)

		drawBorder(
			sd.BorderLeft,
			sd.X-1, sd.Y,
			sd.X-1, sd.Y+seatSize,
		)
		drawBorder(
			sd.BorderRight,
			sd.X+seatSize+1, sd.Y,
			sd.X+seatSize+1, sd.Y+seatSize,
		)
	}

	d.draw.SetColor(color.Black)
	d.draw.SetFontFace(fontUpdated)
	d.draw.DrawStringAnchored(
		share.TimeFormatKr.Replace(now.Format("2006년 1월 2일 Mon pm 3시 4분 기준")),
		float64(d.draw.Width())/2,
		imageUpdatedStringY,
		0.5,
		1,
	)

	d.pngLock.Lock()
	defer d.pngLock.Unlock()

	h := fnv.New128()

	d.pngBodyBuffer.Reset()
	err := pngEncoder.Encode(io.MultiWriter(&d.pngBodyBuffer, h), d.draw.Image())
	if err != nil {
		sentry.CaptureException(err)
		d.pngBody = nil
	}

	d.pngBody = d.pngBodyBuffer.Bytes()
	d.pngBodyLength = strconv.Itoa(len(d.pngBody))
	d.pngETag = hex.EncodeToString(h.Sum(nil))
}
