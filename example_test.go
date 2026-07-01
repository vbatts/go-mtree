package mtree

import (
	"log"
	"os"
)

func ExampleWalkOptions() {
	// Create a spec from a directory, adding sha256 and dropping time.
	dh, err := NewWalkOptions().
		AddKeywords("sha256digest").
		RemoveKeywords("time").
		Walk("/path/to/dir")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := dh.WriteTo(os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func ExampleWalkOptions_Check() {
	// Validate a directory against a previously created spec.
	fh, err := os.Open("/path/to/spec.mtree")
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	spec, err := ParseSpec(fh)
	if err != nil {
		log.Fatal(err)
	}

	deltas, err := NewWalkOptions().
		UseKeywords(spec.UsedKeywords()).
		Check("/path/to/dir", spec)
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range deltas {
		log.Println(d)
	}
}

func ExampleNewTarStreamer() {
	fh, err := os.Open("rootfs.tar")
	if err != nil {
		log.Fatal(err)
	}
	defer fh.Close()

	excludes := []ExcludeFunc{}
	ts := NewTarStreamer(fh, excludes, DefaultTarKeywords)

	dh, err := ts.Hierarchy()
	if err != nil {
		log.Fatal(err)
	}
	_, err =	dh.WriteTo(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
}
