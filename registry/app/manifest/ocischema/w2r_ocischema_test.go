//  Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ocischema

import (
	"encoding/json"
	"testing"

	"github.com/harness/gitness/registry/app/manifest"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

const (
	w2rConfigDgst = "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a"
	w2rLayerDgst  = "sha256:ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	w2rManifestJS = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json",` +
		`"artifactType":"application/vnd.example","config":{"mediaType":"application/vnd.oci.image.config.v1+json",` +
		`"size":2,"digest":"` + w2rConfigDgst + `"},"layers":[` +
		`{"mediaType":"application/vnd.oci.image.layer.v1.tar+gzip","size":3,"digest":"` + w2rLayerDgst + `"},` +
		`{"mediaType":"application/vnd.oci.image.layer.nondistributable.v1.tar+gzip","size":4,` +
		`"digest":"` + w2rLayerDgst + `"}],` +
		`"subject":{"mediaType":"application/vnd.oci.image.manifest.v1+json","size":5,"digest":"` + w2rConfigDgst + `"},` +
		`"annotations":{"k":"v"}}`
	w2rIndexJS = `{"schemaVersion":2,"mediaType":"application/vnd.oci.image.index.v1+json","manifests":[` +
		`{"mediaType":"application/vnd.oci.image.manifest.v1+json","size":7,"digest":"` + w2rConfigDgst + `"}],` +
		`"artifactType":"application/vnd.idx","annotations":{"a":"b"},` +
		`"subject":{"mediaType":"application/vnd.oci.image.manifest.v1+json","size":5,"digest":"` + w2rLayerDgst + `"}}`
)

func TestW2rManifestUnmarshalAndAccessors(t *testing.T) {
	m, desc, err := manifest.UnmarshalManifest(v1.MediaTypeImageManifest, []byte(w2rManifestJS))
	require.NoError(t, err)
	require.Equal(t, v1.MediaTypeImageManifest, desc.MediaType)
	require.Equal(t, int64(len(w2rManifestJS)), desc.Size)
	require.Equal(t, map[string]string{"k": "v"}, desc.Annotations)

	dm, ok := m.(*DeserializedManifest)
	require.True(t, ok)
	require.Len(t, dm.References(), 3, "config plus two layers")
	require.Equal(t, w2rConfigDgst, dm.Config().Digest.String())
	require.Equal(t, dm.Config(), dm.Target())
	require.Len(t, dm.Layers(), 2)
	require.Len(t, dm.DistributableLayers(), 1, "non distributable layers are skipped")
	require.Equal(t, "application/vnd.example", dm.ArtifactType())
	require.Equal(t, int64(5), dm.Subject().Size)
	require.Equal(t, "v", dm.Annotations()["k"])
	require.Equal(t, 2, dm.Version().SchemaVersion)

	mediaType, payload, err := dm.Payload()
	require.NoError(t, err)
	require.Equal(t, v1.MediaTypeImageManifest, mediaType)
	require.JSONEq(t, w2rManifestJS, string(payload))

	out, err := json.Marshal(dm)
	require.NoError(t, err)
	require.JSONEq(t, w2rManifestJS, string(out))
}

func TestW2rManifestDefaults(t *testing.T) {
	dm := &DeserializedManifest{}
	require.Equal(t, v1.MediaTypeImageManifest, dm.Version().MediaType)
	require.Empty(t, dm.Annotations())
	require.Empty(t, dm.Subject().Digest)
	_, err := dm.MarshalJSON()
	require.Error(t, err, "an uninitialised manifest has no canonical JSON")
}

func TestW2rManifestFromStruct(t *testing.T) {
	built, err := FromStruct(Manifest{
		Versioned: SchemaVersion,
		Config:    manifest.Descriptor{MediaType: v1.MediaTypeImageConfig, Size: 1},
	})
	require.NoError(t, err)
	out, err := built.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(out), `"schemaVersion": 2`)
}

func TestW2rManifestRejectsInvalidInput(t *testing.T) {
	_, _, err := manifest.UnmarshalManifest(v1.MediaTypeImageManifest, []byte(`{not json`))
	require.Error(t, err)

	_, _, err = manifest.UnmarshalManifest(v1.MediaTypeImageManifest, []byte(w2rIndexJS))
	require.Error(t, err, "an index is not a manifest")

	wrongType := `{"schemaVersion":2,"mediaType":"application/other","config":{},"layers":[]}`
	_, _, err = manifest.UnmarshalManifest(v1.MediaTypeImageManifest, []byte(wrongType))
	require.Error(t, err)

	var dm DeserializedManifest
	require.Error(t, dm.UnmarshalJSON([]byte(`[]`)))
}

func TestW2rImageIndexUnmarshalAndAccessors(t *testing.T) {
	m, desc, err := manifest.UnmarshalManifest(v1.MediaTypeImageIndex, []byte(w2rIndexJS))
	require.NoError(t, err)
	require.Equal(t, v1.MediaTypeImageIndex, desc.MediaType)
	require.Equal(t, map[string]string{"a": "b"}, desc.Annotations)

	idx, ok := m.(*DeserializedImageIndex)
	require.True(t, ok)
	require.Len(t, idx.References(), 1)
	require.Equal(t, "application/vnd.idx", idx.ArtifactType())
	require.Equal(t, int64(5), idx.Subject().Size)
	require.Equal(t, "b", idx.Annotations()["a"])

	mediaType, payload, err := idx.Payload()
	require.NoError(t, err)
	require.Equal(t, v1.MediaTypeImageIndex, mediaType)
	require.JSONEq(t, w2rIndexJS, string(payload))

	out, err := idx.MarshalJSON()
	require.NoError(t, err)
	require.JSONEq(t, w2rIndexJS, string(out))
}

func TestW2rImageIndexDefaultsAndBuilder(t *testing.T) {
	empty := &DeserializedImageIndex{}
	require.Empty(t, empty.Annotations())
	require.Empty(t, empty.Subject().Digest)
	_, err := empty.MarshalJSON()
	require.Error(t, err)
	mediaType, _, err := empty.Payload()
	require.NoError(t, err)
	require.Equal(t, v1.MediaTypeImageIndex, mediaType)

	built, err := FromDescriptors(
		[]manifest.Descriptor{{MediaType: v1.MediaTypeImageManifest, Size: 1}}, map[string]string{"x": "y"})
	require.NoError(t, err)
	require.Len(t, built.References(), 1)
	require.Equal(t, "y", built.Annotations()["x"])
	out, err := built.MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(out), v1.MediaTypeImageIndex)
}

func TestW2rImageIndexRejectsInvalidInput(t *testing.T) {
	_, _, err := manifest.UnmarshalManifest(v1.MediaTypeImageIndex, []byte(`{not json`))
	require.Error(t, err)

	_, _, err = manifest.UnmarshalManifest(v1.MediaTypeImageIndex, []byte(w2rManifestJS))
	require.Error(t, err, "a manifest is not an index")

	wrongType := `{"schemaVersion":2,"mediaType":"application/other","manifests":[]}`
	_, _, err = manifest.UnmarshalManifest(v1.MediaTypeImageIndex, []byte(wrongType))
	require.Error(t, err)

	var idx DeserializedImageIndex
	require.Error(t, idx.UnmarshalJSON([]byte(`[]`)))
}
