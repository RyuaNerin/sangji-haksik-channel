package share

import (
	"bytes"
	"net/http"
	"strconv"
	"sync"

	skill "github.com/RyuaNerin/go-kakaoskill/v2"
	jsoniter "github.com/json-iterator/go"
)

type SkillData struct {
	hash []byte

	lock sync.RWMutex

	data       []byte
	dataBuffer bytes.Buffer

	text       []byte
	textBuffer bytes.Buffer
}

func (sd *SkillData) GetHash() []byte {
	return sd.hash
}

func (sd *SkillData) CheckHash(h []byte) (changed bool) {
	o := bytes.Equal(sd.hash, h)
	if o {
		return false
	}

	if len(sd.hash) != len(h) {
		sd.hash = make([]byte, len(h))
	}
	copy(sd.hash, h)
	return true
}

func (sd *SkillData) Serve(ctx *skill.Context) bool {
	sd.lock.RLock()
	defer sd.lock.RUnlock()

	if sd.data == nil {
		return false
	}

	ctx.ResponseWriter.WriteHeader(http.StatusOK)
	ctx.ResponseWriter.Write(sd.data)

	return true
}

func (sd *SkillData) ServeHttp(w http.ResponseWriter, r *http.Request) bool {
	sd.lock.RLock()
	defer sd.lock.RUnlock()

	if sd.text == nil {
		return false
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(sd.text)))
	w.WriteHeader(http.StatusOK)
	w.Write(sd.text)

	return true
}

func (sd *SkillData) Update(text []byte, sr *skill.SkillResponse) (err error) {
	sd.lock.Lock()
	defer sd.lock.Unlock()

	sd.data = nil

	sd.dataBuffer.Reset()
	jsoniter.NewEncoder(&sd.dataBuffer).Encode(sr)

	sd.textBuffer.Reset()
	sd.textBuffer.Write(text)

	sd.data = sd.dataBuffer.Bytes()
	sd.text = sd.textBuffer.Bytes()

	return nil
}
