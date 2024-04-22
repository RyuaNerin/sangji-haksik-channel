package share

import (
	"errors"
	"io"
	"os"
	"time"
	"unsafe"

	jsoniter "github.com/json-iterator/go"
)

var Config = func() (r struct {
	Id string `json:"id"`
	Pw string `json:"pw"`

	Fiddler string `json:"fiddler"`

	UpdatePeriodMenu    time.Duration `json:"update-period-menu"`
	UpdatePeriodLibrary time.Duration `json:"update-period-library"`
	UpdatePeriodBus     time.Duration `json:"update-period-bus"`
	UpdatePeriodNotice  time.Duration `json:"update-period-notice"`
},
) {
	fs, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer fs.Close()

	jsoniter.RegisterTypeDecoderFunc(
		"time.Duration",
		func(ptr unsafe.Pointer, iter *jsoniter.Iterator) {
			v := iter.Read()

			switch value := v.(type) {
			case int:
				*((*time.Duration)(ptr)) = time.Duration(value) * time.Second
			case int64:
				*((*time.Duration)(ptr)) = time.Duration(value) * time.Second
			case int32:
				*((*time.Duration)(ptr)) = time.Duration(value) * time.Second

			case string:
				vd, err := time.ParseDuration(value)
				if err != nil {
					panic(err)
				}

				*((*time.Duration)(ptr)) = vd
			default:
				panic(errors.New("type error"))
			}
		},
	)

	err = jsoniter.NewDecoder(fs).Decode(&r)
	if err != nil && err != io.EOF {
		panic(err)
	}

	return r
}()
