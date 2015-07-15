package citadel

import "testing"

func TestParseImageNameTopLevel(t *testing.T) {
	image := "debian:jessie"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "debian" {
		t.Fatalf("expected name debian; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "jessie" {
		t.Fatalf("expected tag jessie; received %s", imageInfo.Tag)
	}
}

func TestParseImageNameTopLevelNoTag(t *testing.T) {
	image := "debian"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "debian" {
		t.Fatalf("expected name debian; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "latest" {
		t.Fatalf("expected tag latest; received %s", imageInfo.Tag)
	}
}

func TestParseImageNamePublic(t *testing.T) {
	image := "citadel/foo:latest"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "citadel/foo" {
		t.Fatalf("expected name citadel/foo; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "latest" {
		t.Fatalf("expected tag latest; received %s", imageInfo.Tag)
	}
}

func TestParseImageNameCustomRegistry(t *testing.T) {
	image := "registry.citadel.com/foo:latest"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "registry.citadel.com/foo" {
		t.Fatalf("expected name registry.citadel.com; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "latest" {
		t.Fatalf("expected tag latest; received %s", imageInfo.Tag)
	}
}

func TestParseImageNameCustomRegistryNoTag(t *testing.T) {
	image := "registry.citadel.com/foo"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "registry.citadel.com/foo" {
		t.Fatalf("expected name registry.citadel.com/foo; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "latest" {
		t.Fatalf("expected tag latest; received %s", imageInfo.Tag)
	}
}

func TestParseImageNameCustomRegistryPort(t *testing.T) {
	image := "registry.citadel.com:49153/foo:bar"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "registry.citadel.com:49153/foo" {
		t.Fatalf("expected name registry.citadel.com:49153/foo; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "bar" {
		t.Fatalf("expected tag bar; received %s", imageInfo.Tag)
	}
}

func TestParseImageNameCustomRegistryPortNamespace(t *testing.T) {
	image := "registry.citadel.com:49153/namespace/foo:bar"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "registry.citadel.com:49153/namespace/foo" {
		t.Fatalf("expected name registry.citadel.com:49153/namespace/foo; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "bar" {
		t.Fatalf("expected tag bar; received %s", imageInfo.Tag)
	}
}

func TestParseImageNameCustomRegistryPortNoTag(t *testing.T) {
	image := "registry.citadel.com:49153/foo"
	imageInfo := ParseImageName(image)
	if imageInfo.Name != "registry.citadel.com:49153/foo" {
		t.Fatalf("expected name registry.citadel.com:49153/foo; received %s", imageInfo.Name)
	}

	if imageInfo.Tag != "latest" {
		t.Fatalf("expected tag latest; received %s", imageInfo.Tag)
	}
}
