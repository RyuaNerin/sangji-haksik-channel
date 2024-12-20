package share

import (
	"encoding/json"
	"io"
	"os"
	"time"
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

	err = json.NewDecoder(fs).Decode(&r)
	if err != nil && err != io.EOF {
		panic(err)
	}

	return r
}()
