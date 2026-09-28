package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// PriceList holds a shop's rates in paise (FSD §9.1). R1a supports A4 only.
type PriceList struct {
	BWOne      int64 `json:"bwOnePaise"`      // per printed side
	BWBoth     int64 `json:"bwBothPaise"`     // per sheet printed on both sides (0 = 2 × one side)
	ColourOne  int64 `json:"colourOnePaise"`  // per printed side (0 = colour not offered)
	ColourBoth int64 `json:"colourBothPaise"` // per sheet (0 = 2 × one side)
	MinCharge  int64 `json:"minChargePaise"`  // per job
	Version    int   `json:"version"`
}

func DefaultPriceList() PriceList {
	return PriceList{BWOne: 200, BWBoth: 300, ColourOne: 1000, ColourBoth: 1800, MinCharge: 0, Version: 1}
}

func (p PriceList) ColourAvailable() bool { return p.ColourOne > 0 }

func (p PriceList) Validate() error {
	if p.BWOne < 50 || p.BWOne > 50000 {
		return fmt.Errorf("%w: B/W one side must be between ₹0.50 and ₹500", ErrValidation)
	}
	for _, v := range []int64{p.BWBoth, p.ColourOne, p.ColourBoth, p.MinCharge} {
		if v < 0 || v > 50000 {
			return fmt.Errorf("%w: prices must be between ₹0 and ₹500", ErrValidation)
		}
	}
	return nil
}

type FileSettings struct {
	Copies    int    `json:"copies"`
	Colour    bool   `json:"colour"`
	BothSides bool   `json:"bothSides"`
	PageRange string `json:"pageRange,omitempty"`
}

func DefaultFileSettings() FileSettings { return FileSettings{Copies: 1} }

var (
	ErrValidation        = errors.New("validation failed")
	ErrPageRange         = errors.New("page_range")
	ErrOptionUnavailable = errors.New("option_unavailable")
)

// ParsePageRange turns "1-3,5" into the selected page numbers (1-based, sorted, unique).
// An empty expression selects every page.
func ParsePageRange(expr string, pages int) ([]int, error) {
	expr = strings.ReplaceAll(strings.TrimSpace(expr), " ", "")
	if pages <= 0 {
		if expr == "" {
			return nil, nil
		}
		return nil, ErrPageRange
	}
	if expr == "" {
		out := make([]int, pages)
		for i := range out {
			out[i] = i + 1
		}
		return out, nil
	}
	seen := map[int]bool{}
	for _, part := range strings.Split(expr, ",") {
		if part == "" {
			return nil, ErrPageRange
		}
		lo, hi := part, part
		if i := strings.Index(part, "-"); i >= 0 {
			lo, hi = part[:i], part[i+1:]
		}
		a, err1 := strconv.Atoi(lo)
		b, err2 := strconv.Atoi(hi)
		if err1 != nil || err2 != nil || a < 1 || b < a || b > pages {
			return nil, ErrPageRange
		}
		for p := a; p <= b; p++ {
			seen[p] = true
		}
	}
	out := make([]int, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Ints(out)
	return out, nil
}

type QuoteLine struct {
	FileID        string `json:"fileId"`
	SelectedPages int    `json:"selectedPages"`
	Sheets        int    `json:"sheets"`
	Sides         int    `json:"sides"`
	Copies        int    `json:"copies"`
	Colour        bool   `json:"colour"`
	BothSides     bool   `json:"bothSides"`
	UnitPaise     int64  `json:"unitPaise"`
	Unit          string `json:"unit"` // side | sheet
	AmountPaise   int64  `json:"amountPaise"`
}

type Quote struct {
	Lines          []QuoteLine `json:"lines"`
	SubtotalPaise  int64       `json:"subtotalPaise"`
	MinChargePaise int64       `json:"minChargePaise"`
	TotalPaise     int64       `json:"totalPaise"`
	PagesTotal     int         `json:"pagesTotal"`
	PagesToConfirm bool        `json:"pagesToConfirm"`
	PriceVersion   string      `json:"priceVersion"`
}

type QuoteFile struct {
	ID       string
	Kind     string // pdf | image
	Pages    int    // 0 = unknown
	Settings FileSettings
}

// NormaliseSettings applies defaults and forces images to one side.
func NormaliseSettings(s FileSettings, kind string) FileSettings {
	if s.Copies == 0 {
		s.Copies = 1
	}
	if kind == "image" {
		s.BothSides = false
		s.PageRange = ""
	}
	s.PageRange = strings.ReplaceAll(strings.TrimSpace(s.PageRange), " ", "")
	return s
}

// ComputeQuote prices a job. All maths is integer paise; only the final total is rounded
// to the nearest rupee, half up (FSD FS-9.3).
func ComputeQuote(pl PriceList, files []QuoteFile) (Quote, error) {
	q := Quote{Lines: make([]QuoteLine, 0, len(files))}
	for _, f := range files {
		s := NormaliseSettings(f.Settings, f.Kind)
		if s.Copies < 1 || s.Copies > 99 {
			return Quote{}, fmt.Errorf("%w: copies must be 1 to 99", ErrValidation)
		}
		if s.Colour && !pl.ColourAvailable() {
			return Quote{}, ErrOptionUnavailable
		}
		line := QuoteLine{FileID: f.ID, Copies: s.Copies, Colour: s.Colour, BothSides: s.BothSides}
		if f.Pages <= 0 {
			q.PagesToConfirm = true
			q.Lines = append(q.Lines, line)
			continue
		}
		sel, err := ParsePageRange(s.PageRange, f.Pages)
		if err != nil {
			return Quote{}, err
		}
		n := len(sel)
		line.SelectedPages = n
		line.Sides = n
		line.Sheets = (n + 1) / 2
		one, both := pl.BWOne, pl.BWBoth
		if s.Colour {
			one, both = pl.ColourOne, pl.ColourBoth
		}
		if s.BothSides {
			if both == 0 {
				both = 2 * one
			}
			line.Unit, line.UnitPaise = "sheet", both
			line.AmountPaise = int64(s.Copies) * int64(line.Sheets) * both
		} else {
			line.Unit, line.UnitPaise = "side", one
			line.Sheets = n
			line.AmountPaise = int64(s.Copies) * int64(n) * one
		}
		q.PagesTotal += n * s.Copies
		q.SubtotalPaise += line.AmountPaise
		q.Lines = append(q.Lines, line)
	}
	total := q.SubtotalPaise
	if total > 0 && total < pl.MinCharge {
		total = pl.MinCharge
		q.MinChargePaise = pl.MinCharge
	}
	q.TotalPaise = ((total + 50) / 100) * 100
	q.PriceVersion = priceVersion(pl, files)
	return q, nil
}

func priceVersion(pl PriceList, files []QuoteFile) string {
	type f struct {
		ID    string
		Pages int
		S     FileSettings
	}
	arr := make([]f, len(files))
	for i, x := range files {
		arr[i] = f{x.ID, x.Pages, NormaliseSettings(x.Settings, x.Kind)}
	}
	b, _ := json.Marshal(struct {
		P PriceList
		F []f
	}{pl, arr})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:12]
}

// FileKind classifies a MIME type.
func FileKind(mime string) string {
	if strings.HasPrefix(mime, "image/") {
		return "image"
	}
	return "pdf"
}

var AllowedMimes = map[string]bool{
	"application/pdf": true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/heic":      true,
	"image/webp":      true,
}
