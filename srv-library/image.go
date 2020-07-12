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
)

const (
	imageUpdatedStringX = 10
	imageUpdatedStringY = 10

	seatUlY      = 5
	seatUlX      = 5
	seatUlWidth  = 5
	seatUlMargin = 5

	exitKey  = "exit"
	seatSize = 38

	colorSeat  = "#f5f5f5"
	colorUsing = "#9fd7e7"
)

var (
	imgDisabled   = mustLoadLoage("explanatory_type_line2.png")
	imgWheelChair = mustLoadLoage("explanatory_type_line2.png")
	imgNotebook   = mustLoadLoage("explanatory_type_line2.png")
	imgExit       = mustLoadLoage("img_exit.png")
	imgPillar     = mustLoadLoage("img_pillar.png")

	pngEncoder = png.Encoder{
		CompressionLevel: png.BestCompression,
		BufferPool:       new(pngBufferPool),
	}
)

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

type imageData struct {
	seatInfo map[string]seatDrawingStyle

	draw *gg.Context

	background image.Image

	pngLock       sync.RWMutex
	pngBody       []byte
	pngBodyBuffer bytes.Buffer
	pngBodyLength string
	pngETag       string
}

type seatDrawingStyle struct {
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
		Background   string                      `json:"background"`
		Width        int                         `json:"width"`
		Height       int                         `json:"height"`
		SeatLocation map[string]seatDrawingStyle `json:"seat-location"`
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

	data = &imageData{
		seatInfo:   drawingData.SeatLocation,
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
	if seatMap == nil {
		d.pngLock.Lock()
		defer d.pngLock.Unlock()

		d.pngBody = nil

		return
	}

	d.draw.SetColor(color.White)
	d.draw.DrawRectangle(0, 0, float64(d.draw.Width()), float64(d.draw.Height()))
	d.draw.DrawImage(d.background, 0, 0)

	// exit
	sd, ok := d.seatInfo[exitKey]
	if ok {
		d.draw.DrawImage(imgExit, sd.X, sd.Y)
	}

	// pillar
	for k, sd := range d.seatInfo {
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
		sd, ok := d.seatInfo[seat.SeatNum]
		if !ok {
			continue
		}

		// 좌석 네모 그리기
		if seat.Using {
			d.draw.SetHexColor(colorSeat)
		} else {
			d.draw.SetHexColor(colorUsing)
		}
		d.draw.DrawRectangle(float64(sd.X), float64(sd.Y), seatSize, seatSize)
		d.draw.Fill()

		if seat.Disabled {
			d.draw.DrawImage(imgDisabled, 0, 0)
		}

		// 왼쪽 상단에 휠체어 노트북 그리기
		ulX := sd.X + seatUlX
		if seat.WheelChair {
			d.draw.DrawImage(imgWheelChair, ulX, sd.Y+seatUlY)
			ulX += seatUlWidth + seatUlMargin
		}
		if seat.NoteBook {
			d.draw.DrawImage(imgNotebook, ulX, sd.Y+seatUlY)
		}

		d.draw.SetColor(color.Black)
		tw, th := d.draw.MeasureString(seat.SeatNum)
		d.draw.DrawString(
			seat.SeatNum,
			float64(sd.X)+(float64(seatSize)-tw)/2,
			float64(sd.Y)+(float64(seatSize)-th)/2,
		)

		drawBorder(
			sd.BorderTop,
			sd.X, sd.Y,
			sd.X+seatSize, sd.Y,
		)
		drawBorder(
			sd.BorderBottom,
			sd.X, sd.Y+seatSize,
			sd.X+seatSize, sd.Y+seatSize,
		)

		drawBorder(
			sd.BorderLeft,
			sd.X, sd.Y,
			sd.X, sd.Y+seatSize,
		)
		drawBorder(
			sd.BorderRight,
			sd.X+seatSize, sd.Y,
			sd.X+seatSize, sd.Y+seatSize,
		)
	}

	d.draw.DrawString(
		share.TimeFormatKr.Replace(now.Format("2006년 1월 2일 Mon pm 3시 4분 기준")),
		imageUpdatedStringX,
		imageUpdatedStringY,
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
