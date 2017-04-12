package updatedetector

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/datastore"
)

type URLList struct {
	URLs   []string
	UserID string
}

func NewURLListKey(ctx context.Context, userId string) *datastore.Key {
	return datastore.NewKey(ctx, "URLLIST", userId, 0, nil)
}

func GetAllURLList(ctx context.Context, urllist *[]URLList) ([]*datastore.Key, error) {
	return datastore.NewQuery("URLLIST").GetAll(ctx, urllist)
}

type HTML struct {
	URL  string
	HTML []byte
}

func NewHTMLKey(ctx context.Context, userID, url string) *datastore.Key {
	return datastore.NewKey(ctx, "HTMLTEXT", userID+":"+url, 0, nil)
}
