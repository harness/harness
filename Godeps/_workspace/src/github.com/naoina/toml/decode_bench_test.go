package toml_test

import (
	"testing"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/naoina/toml"
)

func BenchmarkUnmarshal(b *testing.B) {
	var v struct {
		Title string
		Owner struct {
			Name         string
			Organization string
			Bio          string
			Dob          time.Time
		}
		Database struct {
			Server        string
			Ports         []int
			ConnectionMax int
			Enabled       bool
		}
		Servers struct {
			Alpha struct {
				IP string
				DC string
			}
			Beta struct {
				IP string
				DC string
			}
		}
		Clients struct {
			Data  []interface{}
			Hosts []string
		}
	}
	data, err := loadTestData()
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		if err := toml.Unmarshal(data, &v); err != nil {
			b.Fatal(err)
		}
	}
}
