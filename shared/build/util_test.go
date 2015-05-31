package build

import "testing"

func TestParseImageName(t *testing.T) {
	images := []struct {
		owner string
		name  string
		tag   string
		cname string
	}{
		// full image name with all 3 sections present
		{"johnsmith", "redis", "2.8", "johnsmith/redis:2.8"},
		// image name with no tag specified
		{"johnsmith", "redis", "latest", "johnsmith/redis"},
		// image name with no owner specified
		{"bradrydzewski", "redis", "2.8", "redis:2.8"},
		// image name with hostname
		{"docker.example.com/johnsmith", "redis", "latest", "docker.example.com/johnsmith/redis"},
		// image name with ownly name specified
		{"bradrydzewski", "redis2", "latest", "redis2"},
		// image name that is a known alias
		{"relateiq", "cassandra", "latest", "cassandra"},
	}

	for _, img := range images {
		owner, name, tag := parseImageName(img.cname)
		if owner != img.owner {
			t.Errorf("Expected image %s with owner %s, got %s", img.cname, img.owner, owner)
		}
		if name != img.name {
			t.Errorf("Expected image %s with name %s, got %s", img.cname, img.name, name)
		}
		if tag != img.tag {
			t.Errorf("Expected image %s with tag %s, got %s", img.cname, img.tag, tag)
		}
	}
}
