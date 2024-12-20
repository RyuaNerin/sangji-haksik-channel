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

	UpdatePeriodMenu    Duration `json:"update-period-menu"`
	UpdatePeriodLibrary Duration `json:"update-period-library"`
	UpdatePeriodBus     Duration `json:"update-period-bus"`
	UpdatePeriodNotice  Duration `json:"update-period-notice"`
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

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}

	t, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(t)
	return nil
}

func (d Duration) Value() time.Duration {
	return time.Duration(d)
}
