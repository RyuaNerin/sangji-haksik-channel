package share

import (
	"bytes"
	"io"
	"net/http"
	"sync"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	"github.com/getsentry/sentry-go"
	jsoniter "github.com/json-iterator/go"
)

type SkillData struct {
	hash []byte

	lock       sync.RWMutex
	data       []byte
	dataBuffer bytes.Buffer

	dataBufferTemp bytes.Buffer
}

func (sd *SkillData) GetHash() []byte {
	return sd.hash
}

func (sd *SkillData) CheckHash(h []byte) (changed bool) {
	o := bytes.Equal(sd.hash, h)
	if o {
		return false
	}

	if len(sd.data) != len(h) {
		sd.data = make([]byte, len(h))
	}
	copy(sd.data, h)
	return true
}

func (sd *SkillData) Serve(ctx *skill.Context) bool {
	sd.lock.RLock()
	defer sd.lock.RUnlock()

	if sd.data == nil {
		return false
	} else {
		ctx.ResponseWriter.WriteHeader(http.StatusOK)
		ctx.ResponseWriter.Write(sd.data)
	}

	return true
}

func (sd *SkillData) Update(sr *skill.SkillResponse) (err error) {
	sd.dataBufferTemp.Reset()
	err = jsoniter.NewEncoder(&sd.dataBufferTemp).Encode(sr)
	if err != nil {
		sentry.CaptureException(err)
		return err
	}

	sd.lock.Lock()
	defer sd.lock.Unlock()

	sd.data = nil

	sd.dataBuffer.Reset()
	io.Copy(&sd.dataBuffer, &sd.dataBufferTemp)

	sd.data = sd.dataBuffer.Bytes()

	return nil
}
