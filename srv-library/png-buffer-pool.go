package srvlibrary

import (
	"image/png"
	"sync"
)

type pngBufferPool struct {
	p sync.Pool
}

func (p *pngBufferPool) Get() *png.EncoderBuffer {
	return p.p.Get().(*png.EncoderBuffer)
}

func (p *pngBufferPool) Put(b *png.EncoderBuffer) {
	p.p.Put(b)
}
