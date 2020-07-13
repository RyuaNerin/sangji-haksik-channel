package srvlibrary

import (
	"image/png"
	"sync"
)

type pngBufferPool struct {
	p sync.Pool
}

func (p *pngBufferPool) Get() *png.EncoderBuffer {
	pp, _ := p.p.Get().(*png.EncoderBuffer)
	return pp
}

func (p *pngBufferPool) Put(b *png.EncoderBuffer) {
	p.p.Put(b)
}
