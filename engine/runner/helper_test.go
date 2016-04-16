package runner

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/franela/goblin"
)

func TestHelper(t *testing.T) {
	g := goblin.Goblin(t)

	g.Describe("Parsing", func() {

		g.It("should unmarhsal file []byte", func() {
			res, err := Parse(sample)
			if err != nil {
				t.Error(err)
				return
			}
			g.Assert(err == nil).IsTrue("expect file parsed")
			g.Assert(len(res.Containers)).Equal(2)
			g.Assert(len(res.Volumes)).Equal(1)
		})

		g.It("should unmarshal from file", func() {
			temp, _ := ioutil.TempFile("", "spec_")
			defer os.Remove(temp.Name())

			ioutil.WriteFile(temp.Name(), sample, 0700)

			_, err := ParseFile(temp.Name())
			if err != nil {
				t.Error(err)
				return
			}
			g.Assert(err == nil).IsTrue("expect file parsed")
		})

		g.It("should error when file not found", func() {
			_, err := ParseFile("/tmp/foo/bar/dummy/file.json")
			g.Assert(err == nil).IsFalse("expect file not found error")
		})
	})
}

// invalid json representation, simulate parsing error
var invalid = []byte(`[]`)

// valid json representation, verify parsing
var sample = []byte(`{
	"containers": [
		{
			"name": "container_0",
			"image": "node:latest" 
		},
		{
			"name": "container_1",
			"image": "golang:latest" 
		}
	],
	"volumes": [
		{
			"name": "volume_0"
		}
	],
	"program": {
		"type": "list",
		"body": [
			{
				"type": "defer",
				"body": {
					"type": "recover",
					"body": {
						"type": "run",
						"name": "container_0"
					}
				},
				"defer": {
					"type": "parallel",
					"body": [
						{
							"type": "run",
							"name": "container_1"
						},
						{
							"type": "run",
							"name": "container_1"
						}
					],
					"limit": 2
				}
			}
		]
	}
}`)
