package pagination

const (
	DefaultLimit = 20
	MaxLimit     = 100
)

type Params struct {
	Page  int
	Limit int
}

type Result struct {
	Page  int `json:"page"`
	Limit int `json:"limit"`
	Total int `json:"total"`
}

func Normalize(page, limit int) Params {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return Params{Page: page, Limit: limit}
}

func (p Params) Offset() int {
	return (p.Page - 1) * p.Limit
}
