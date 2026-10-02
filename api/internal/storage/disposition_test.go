package storage

import (
	"mime"
	"testing"
)

func TestContentDisposition(t *testing.T) {
	cases := []struct{ disp, in, want string }{
		{"attachment", "report.pdf", `attachment; filename="report.pdf"`},
		{"inline", `a"b.pdf`, `inline; filename="ab.pdf"`},
		{"attachment", "क्रीडांgan (1).png", `attachment; filename="_______gan (1).png"; filename*=UTF-8''%E0%A4%95%E0%A5%8D%E0%A4%B0%E0%A5%80%E0%A4%A1%E0%A4%BE%E0%A4%82gan%20%281%29.png`},
		{"attachment", "", `attachment; filename="file"`},
	}
	for _, c := range cases {
		got := ContentDisposition(c.disp, c.in)
		if got != c.want {
			t.Errorf("%q:\n got %s\nwant %s", c.in, got, c.want)
		}
		for i := 0; i < len(got); i++ {
			if got[i] > 0x7e {
				t.Fatalf("%q: header not ASCII: %s", c.in, got)
			}
		}
		// Browsers (and Go) read the UTF-8 name back from filename*.
		if _, params, err := mime.ParseMediaType(got); err != nil || params["filename"] != safeFilename(c.in) && c.in != "" {
			t.Errorf("%q: parsed back %q (%v)", c.in, params["filename"], err)
		}
	}
}
