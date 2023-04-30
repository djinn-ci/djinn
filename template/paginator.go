package template

import "djinn-ci.com/database"

func NewPaginator[M database.Model](page *Page, p *database.Paginator[M]) *Paginator {
	return &Paginator{
		Page:  page,
		Pages: p.Pages,
		Prev:  p.Prev(),
		Curr:  p.Page(),
		Next:  p.Next(),
		Query: page.URL.Query(),
	}
}
