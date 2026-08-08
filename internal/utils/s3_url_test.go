package utils

import "testing"

func TestBuildS3ObjectURL(t *testing.T) {
	t.Parallel()

	got := BuildS3ObjectURL("my-bucket", "ap-southeast-1", "images/a b.jpg", "", false)
	want := "https://my-bucket.s3.ap-southeast-1.amazonaws.com/images/a%20b.jpg"
	if got != want {
		t.Fatalf("virtual-hosted: got %q want %q", got, want)
	}

	got = BuildS3ObjectURL("my-bucket", "ap-southeast-1", "images/a.jpg", "http://localhost:9000", true)
	want = "http://localhost:9000/my-bucket/images/a.jpg"
	if got != want {
		t.Fatalf("custom endpoint: got %q want %q", got, want)
	}

	got = BuildS3ObjectURL("my-bucket", "eu-west-1", "k", "", true)
	want = "https://s3.eu-west-1.amazonaws.com/my-bucket/k"
	if got != want {
		t.Fatalf("path-style AWS: got %q want %q", got, want)
	}
}
