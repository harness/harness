package parse

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
)

func TestUnmarshal(t *testing.T) {

	node1 := NewRunNode().SetName("foo")
	node2 := NewRecoverNode().SetBody(node1)

	node3 := NewRunNode().SetName("bar")
	node4 := NewRunNode().SetName("bar")

	node5 := NewParallelNode().
		Append(node3).
		Append(node4).
		SetLimit(2)

	node6 := NewDeferNode().
		SetBody(node2).
		SetDefer(node5)

	tree := NewTree()
	tree.Append(node6)

	encoded, err := json.MarshalIndent(tree, "", "\t")
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(encoded, sample) {
		t.Errorf("Want to marshal Tree to %s, got %s",
			string(sample),
			string(encoded),
		)
	}

	parsed, err := Parse(encoded)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(tree, parsed) {
		t.Errorf("Want to marsnal and then unmarshal Tree")
	}
}

var sample = []byte(`{
	"type": "list",
	"body": [
		{
			"type": "defer",
			"body": {
				"type": "recover",
				"body": {
					"type": "run",
					"name": "foo"
				}
			},
			"defer": {
				"type": "parallel",
				"body": [
					{
						"type": "run",
						"name": "bar"
					},
					{
						"type": "run",
						"name": "bar"
					}
				],
				"limit": 2
			}
		}
	]
}`)
