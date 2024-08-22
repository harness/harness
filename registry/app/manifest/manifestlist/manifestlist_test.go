// Source: https://github.com/distribution/distribution

// Copyright 2014 https://github.com/distribution/distribution Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package manifestlist

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/harness/gitness/registry/app/manifest"
	"github.com/harness/gitness/registry/app/manifest/schema2"

	v1 "github.com/opencontainers/image-spec/specs-go/v1"
)

const expectedManifestListSerialization = `{
   "schemaVersion": 2,
   "mediaType": "application/vnd.docker.distribution.manifest.list.v2+json",
   "manifests": [
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "digest": "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b",
         "size": 985,
         "platform": {
            "architecture": "amd64",
            "os": "linux",
            "features": [
               "sse4"
            ]
         }
      },
      {
         "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
         "digest": "sha256:6346340964309634683409684360934680934608934608934608934068934608",
         "size": 2392,
         "platform": {
            "architecture": "sun4m",
            "os": "sunos"
         }
      }
   ]
}`

func makeTestManifestList(t *testing.T, mediaType string) ([]ManifestDescriptor, *DeserializedManifestList) {
	manifestDescriptors := []ManifestDescriptor{
		{
			Descriptor: manifest.Descriptor{
				MediaType: "application/vnd.docker.distribution.manifest.v2+json",
				Digest:    "sha256:1a9ec845ee94c202b2d5da74a24f0ed2058318bfa9879fa541efaecba272e86b",
				Size:      985,
			},
			Platform: PlatformSpec{
				Architecture: "amd64",
				OS:           "linux",
				Features:     []string{"sse4"},
			},
		},
		{
			Descriptor: manifest.Descriptor{
				MediaType: "application/vnd.docker.distribution.manifest.v2+json",
				Digest:    "sha256:6346340964309634683409684360934680934608934608934608934068934608",
				Size:      2392,
			},
			Platform: PlatformSpec{
				Architecture: "sun4m",
				OS:           "sunos",
			},
		},
	}

	deserialized, err := fromDescriptorsWithMediaType(manifestDescriptors, mediaType)
	if err != nil {
		t.Fatalf("error creating DeserializedManifestList: %v", err)
	}

	return manifestDescriptors, deserialized
}

func TestManifestList(t *testing.T) {
	manifestDescriptors, deserialized := makeTestManifestList(t, MediaTypeManifestList)
	mediaType, canonical, _ := deserialized.Payload()

	if mediaType != MediaTypeManifestList {
		t.Fatalf("unexpected media type: %s", mediaType)
	}

	// Check that the canonical field is the same as json.MarshalIndent
	// with these parameters.
	expected, err := json.MarshalIndent(&deserialized.ManifestList, "", "   ")
	if err != nil {
		t.Fatalf("error marshaling manifest list: %v", err)
	}
	if !bytes.Equal(expected, canonical) {
		t.Fatalf("manifest bytes not equal:\nexpected:\n%s\nactual:\n%s\n", string(expected), string(canonical))
	}

	// Check that the canonical field has the expected value.
	if !bytes.Equal([]byte(expectedManifestListSerialization), canonical) {
		t.Fatalf(
			"manifest bytes not equal:\nexpected:\n%s\nactual:\n%s\n",
			expectedManifestListSerialization,
			string(canonical),
		)
	}

	var unmarshalled DeserializedManifestList
	if err := json.Unmarshal(deserialized.canonical, &unmarshalled); err != nil {
		t.Fatalf("error unmarshaling manifest: %v", err)
	}

	if !reflect.DeepEqual(&unmarshalled, deserialized) {
		t.Fatalf("manifests are different after unmarshaling: %v != %v", unmarshalled, *deserialized)
	}

	references := deserialized.References()
	if len(references) != 2 {
		t.Fatalf("unexpected number of references: %d", len(references))
	}
	for i := range references {
		platform := manifestDescriptors[i].Platform
		expectedPlatform := &v1.Platform{
			Architecture: platform.Architecture,
			OS:           platform.OS,
			OSFeatures:   platform.OSFeatures,
			OSVersion:    platform.OSVersion,
			Variant:      platform.Variant,
		}
		if !reflect.DeepEqual(references[i].Platform, expectedPlatform) {
			t.Fatalf("unexpected value %d returned by References: %v", i, references[i])
		}
		references[i].Platform = nil
		if !reflect.DeepEqual(references[i], manifestDescriptors[i].Descriptor) {
			t.Fatalf("unexpected value %d returned by References: %v", i, references[i])
		}
	}
}

func mediaTypeTest(contentType string, mediaType string, shouldError bool) func(*testing.T) {
	return func(t *testing.T) {
		var m *DeserializedManifestList
		_, m = makeTestManifestList(t, mediaType)

		_, canonical, err := m.Payload()
		if err != nil {
			t.Fatalf("error getting payload, %v", err)
		}

		unmarshalled, descriptor, err := manifest.UnmarshalManifest(
			contentType,
			canonical,
		)

		if shouldError {
			if err == nil {
				t.Fatalf("bad content type should have produced error")
			}
			return
		}
		if err != nil {
			t.Fatalf("error unmarshaling manifest, %v", err)
		}

		asManifest, ok := unmarshalled.(*DeserializedManifestList)
		if !ok {
			t.Fatalf("Error: unmarshalled is not of type *DeserializedManifestLis")
			return
		}
		if asManifest.MediaType != mediaType {
			t.Fatalf("Bad media type '%v' as unmarshalled", asManifest.MediaType)
		}

		if descriptor.MediaType != contentType {
			t.Fatalf("Bad media type '%v' for descriptor", descriptor.MediaType)
		}

		unmarshalledMediaType, _, _ := unmarshalled.Payload()
		if unmarshalledMediaType != contentType {
			t.Fatalf("Bad media type '%v' for payload", unmarshalledMediaType)
		}
	}
}

func TestMediaTypes(t *testing.T) {
	t.Run("ManifestList_No_MediaType", mediaTypeTest(MediaTypeManifestList, "", true))
	t.Run("ManifestList", mediaTypeTest(MediaTypeManifestList, MediaTypeManifestList, false))
	t.Run("ManifestList_Bad_MediaType", mediaTypeTest(MediaTypeManifestList, MediaTypeManifestList+"XXX", true))
}

func TestValidateManifestList(t *testing.T) {
	man := schema2.Manifest{
		Config: manifest.Descriptor{Size: 1},
		Layers: []manifest.Descriptor{{Size: 2}},
	}
	manifestList := ManifestList{
		Manifests: []ManifestDescriptor{
			{Descriptor: manifest.Descriptor{Size: 3}},
		},
	}
	t.Run(
		"valid", func(t *testing.T) {
			b, err := json.Marshal(manifestList)
			if err != nil {
				t.Fatal("unexpected error marshaling manifest list", err)
			}
			if err := validateManifestList(b); err != nil {
				t.Error("list should be valid", err)
			}
		},
	)
	t.Run(
		"invalid", func(t *testing.T) {
			b, err := json.Marshal(man)
			if err != nil {
				t.Fatal("unexpected error marshaling manifest", err)
			}
			if err := validateManifestList(b); err == nil {
				t.Error("manifest should not be valid")
			}
		},
	)
}
