package data

// all of our models

type Link struct {
	ID      string
	Short   string
	FullURL string
	Active  bool
}

type LinkStore interface {
	SaveLink(link Link) error
	GetLinkByID(id string) (*Link, error)
	GetLinkByShort(short string) (*Link, error)
}
