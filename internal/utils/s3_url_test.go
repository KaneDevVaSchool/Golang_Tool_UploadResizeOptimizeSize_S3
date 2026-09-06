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

func TestParseS3ObjectKey(t *testing.T) {
	t.Parallel()

	bucket := "my-bucket"
	cases := []struct {
		url  string
		want string
	}{
		{
			"https://my-bucket.s3.ap-southeast-1.amazonaws.com/images/a%20b.jpg",
			"images/a b.jpg",
		},
		{
			"https://s3.eu-west-1.amazonaws.com/my-bucket/k",
			"k",
		},
		{
			"http://localhost:9000/my-bucket/vaschools-uploads/tranh_thumb.webp",
			"vaschools-uploads/tranh_thumb.webp",
		},
		{"", ""},
		{"https://other-bucket.s3.amazonaws.com/x.jpg", ""},
	}
	for _, tc := range cases {
		got := ParseS3ObjectKey(tc.url, bucket)
		if got != tc.want {
			t.Fatalf("ParseS3ObjectKey(%q): got %q want %q", tc.url, got, tc.want)
		}
	}
}
