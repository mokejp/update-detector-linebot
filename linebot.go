package updatedetector

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"golang.org/x/text/unicode/norm"

	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	aelog "google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/line/line-bot-sdk-go/linebot/httphandler"
)

const (
	ctxKeyBotClient = "bot-client"

	cmdTextList   = "一覧"
	cmdTextDelete = "削除"

	httpUserAgent = "Agent:Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/56.0.2924.87 Safari/537.36"
)

func botClient(ctx context.Context) *linebot.Client {
	return ctx.Value(ctxKeyBotClient).(*linebot.Client)
}

type UpdateDetectorBot struct {
	webhookHandler *httphandler.WebhookHandler
}

func NewUpdateDetectorBot(channelSecret, channelToken string) (*UpdateDetectorBot, error) {
	handler, err := httphandler.New(channelSecret, channelToken)
	if err != nil {
		return nil, err
	}
	bot := &UpdateDetectorBot{
		webhookHandler: handler,
	}
	handler.HandleEvents(bot.handleEvents)
	return bot, nil
}

func (bot *UpdateDetectorBot) WebHook(w http.ResponseWriter, r *http.Request) {
	bot.webhookHandler.ServeHTTP(w, r)
}

func (bot *UpdateDetectorBot) newContext(req *http.Request) (context.Context, error) {
	ctx := appengine.NewContext(req)
	httpclient := urlfetch.Client(ctx)
	client, err := bot.webhookHandler.NewClient(linebot.WithHTTPClient(httpclient))
	if err != nil {
		return nil, err
	}
	ctx = context.WithValue(ctx, ctxKeyBotClient, client)
	return ctx, nil
}

func (bot *UpdateDetectorBot) handleEvents(events []*linebot.Event, req *http.Request) {
	ctx, err := bot.newContext(req)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
	for _, event := range events {
		switch event.Type {
		case linebot.EventTypeMessage:
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				bot.handleMessageText(ctx, event, message)
			}
		}

	}
}

func (bot *UpdateDetectorBot) handleMessageText(ctx context.Context, event *linebot.Event, message *linebot.TextMessage) {
	text := string(norm.NFKC.Bytes([]byte(message.Text)))
	url, err := url.Parse(text)
	if err == nil && url.Scheme != "" && url.Host != "" {
		// URL 登録
		bot.handleAddURL(ctx, event, url)
		return
	} else if strings.HasPrefix(text, cmdTextList) {
		bot.handleURLList(ctx, event)
	} else if strings.HasPrefix(text, cmdTextDelete) {
		number := strings.TrimSpace(strings.TrimLeft(text, cmdTextDelete))
		bot.handleDeleteURL(ctx, event, number)
	} else {
		bot.replyUsage(ctx, event)
	}
}

func (bot *UpdateDetectorBot) replyUsage(ctx context.Context, event *linebot.Event) {
	client := botClient(ctx)
	message := "監視したいURLを送ってね！\n「一覧」と送ると、現在監視している一覧、「削除 [番号]」で番号で指定したURLを消せるよ！"
	_, err := client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message)).WithContext(ctx).Do()
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
}

func (bot *UpdateDetectorBot) getListMessage(urls []string) string {
	message := ""
	if len(urls) == 0 {
		message = "URLが登録されていません"
	} else {
		for i, url := range urls {
			message += fmt.Sprintf("%d: %s\n", i+1, url)
		}
	}
	return strings.TrimSpace(message)
}

func (bot *UpdateDetectorBot) handleURLList(ctx context.Context, event *linebot.Event) {
	client := botClient(ctx)
	urlList := new(URLList)

	key := NewURLListKey(ctx, event.Source.UserID)
	err := datastore.Get(ctx, key, urlList)
	if err == datastore.ErrNoSuchEntity {
	} else if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
	message := bot.getListMessage(urlList.URLs)
	_, err = client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message)).WithContext(ctx).Do()
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
}

func (bot *UpdateDetectorBot) handleAddURL(ctx context.Context, event *linebot.Event, url *url.URL) {
	client := botClient(ctx)
	urlList := new(URLList)

	key := NewURLListKey(ctx, event.Source.UserID)
	err := datastore.Get(ctx, key, urlList)
	if err == datastore.ErrNoSuchEntity {
	} else if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
	exists := false
	for _, u := range urlList.URLs {
		if u == url.String() {
			exists = true
		}
	}
	var message string
	if !exists {
		urlList.URLs = append(urlList.URLs, url.String())
		urlList.UserID = event.Source.UserID
		_, err = datastore.Put(ctx, key, urlList)
		if err != nil {
			aelog.Errorf(ctx, "%v", err)
			return
		}
		count := len(urlList.URLs)
		message = fmt.Sprintf("%d: %s を登録しました", count, url.String())
	} else {
		message = "このURLは既に登録済みです"
	}
	_, err = client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message)).WithContext(ctx).Do()
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
}

func (bot *UpdateDetectorBot) handleDeleteURL(ctx context.Context, event *linebot.Event, number string) {
	client := botClient(ctx)
	urlList := new(URLList)

	idx, err := strconv.Atoi(number)
	if err != nil {
		message := "削除1 のように入力してください"
		_, err = client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message)).WithContext(ctx).Do()
		if err != nil {
			aelog.Errorf(ctx, "%v", err)
			return
		}
	}
	key := NewURLListKey(ctx, event.Source.UserID)
	err = datastore.Get(ctx, key, urlList)
	if err == datastore.ErrNoSuchEntity {
	} else if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
	if idx > len(urlList.URLs) {
		message := "登録されていない番号です"
		_, err = client.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message)).WithContext(ctx).Do()
		if err != nil {
			aelog.Errorf(ctx, "%v", err)
			return
		}
		return
	}
	diffKey := NewHTMLKey(ctx, event.Source.UserID, urlList.URLs[idx-1])
	datastore.Delete(ctx, diffKey)
	message := fmt.Sprintf("%s を削除しました\n", urlList.URLs[idx-1])
	urlList.URLs = deleteSlice(urlList.URLs, idx-1)
	_, err = datastore.Put(ctx, key, urlList)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
	messageList := bot.getListMessage(urlList.URLs)
	_, err = client.ReplyMessage(event.ReplyToken,
		linebot.NewTextMessage(message),
		linebot.NewTextMessage(messageList)).WithContext(ctx).Do()
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
}

func (bot *UpdateDetectorBot) CronHook(w http.ResponseWriter, r *http.Request) {
	ctx, err := bot.newContext(r)

	var urlLists []URLList
	_, err = GetAllURLList(ctx, &urlLists)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return
	}
	var wg sync.WaitGroup
	for _, urlList := range urlLists {
		for _, url := range urlList.URLs {
			wg.Add(1)
			go func(userID, url string) {
				checkUpdate(ctx, userID, url, r)
				wg.Done()
			}(urlList.UserID, url)
		}
	}
	wg.Wait()
}

func checkUpdate(ctx context.Context, userId, url string, r *http.Request) error {
	httpclient := urlfetch.Client(ctx)
	client := botClient(ctx)
	diffKey := NewHTMLKey(ctx, userId, url)
	html := new(HTML)
	err := datastore.Get(ctx, diffKey, html)
	exists := true
	if err == datastore.ErrNoSuchEntity {
		exists = false
	} else if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return err
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return err
	}
	req.Header.Set("User-Agent", httpUserAgent)
	resp, err := httpclient.Do(req)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		aelog.Warningf(ctx, "Request Error HTTP %d: %s", resp.StatusCode, url)
		return nil
	}
	body := HTMLToBytes(ctx, resp.Body, resp.Header.Get("Content-Type"))
	aelog.Debugf(ctx, url)
	if exists {
		if !bytes.Equal(html.HTML, body) {
			aelog.Debugf(ctx, string(html.HTML))
			aelog.Debugf(ctx, string(body))
			diffText := []rune(diffText(html.HTML, body))
			aelog.Debugf(ctx, string(diffText))
			if len(diffText) > 200 {
				diffText = append(diffText[0:200], []rune("\n...(省略)")...)
			}
			message := fmt.Sprintf("%s が更新されました", html.URL)
			_, err = client.PushMessage(
				userId,
				linebot.NewTextMessage(message),
				linebot.NewTextMessage(string(diffText))).WithContext(ctx).Do()
			if err != nil {
				aelog.Errorf(ctx, "%v", err)
				return err
			}
		}
	}
	html.URL = url
	html.HTML = body
	_, err = datastore.Put(ctx, diffKey, html)
	if err != nil {
		aelog.Errorf(ctx, "%v", err)
		return err
	}
	return nil
}
