package helix

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var twitchAuthCh chan bool
var twitchAuth bool
var twitchAccessToken string
var twitchAppAccessToken string
var twitchRefreshToken string
var followEventSubscriptions []func(string)

type twitchAuthResp struct {
	Access_token  string
	Refresh_token string
	Expires_in    int
	Scope         []string
	Token_type    string
}

type TwitchUser struct {
	UserID        string `json:"id"`
	DisplayName   string `json:"display_name"`
	ProfileImgURL string `json:"profile_image_url"`
	ChannelDesc   string `json:"description"`
}

type ChannelInfo struct {
	BroadcasterID       string `json:"broadcaster_id"`
	BroadcasterName     string `json:"broadcaster_name"`
	GameName            string `json:"game_name"`
	GameID              string `json:"game_id"`
	BroadcasterLanguage string `json:"broadcaster_language"`
	Title               string
}

type EventSubscription struct {
	ID        string
	Status    string
	Type      string
	Version   string
	Cost      int
	Condition struct {
		BroadCasterUserID string `json:"broadcaster_user_id"`
	}
	CreatedAt time.Time `json:"created_at"`
	Transport struct {
		Method   string
		Callback string
	}
}

func Init() {
	go func() {
		twitchAuthCh = make(chan bool)
		
		twitchAuth = <-twitchAuthCh
		fmt.Println("Auth'd for twitch api")
		twitchAppAccessToken = getAppAccessToken()

		// Get active eventsubs cancel them since they likely have an out of date callback url
		eventSubs := getSubscriptions()
		var alreadySubbed bool
		for _, es := range eventSubs {
			if es.Transport.Callback == os.Getenv("TWITCH_CALLBACK_URL") {
				alreadySubbed = true
				fmt.Println("Eventsub already active. Move along.")
				continue
			}
			deleteSubscription(es.ID)
		}
		if !alreadySubbed {
			subscribe("channel.follow")
		}
	}()
}


func TwitchAuthCb(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	code := r.FormValue("code")
	if code == "" {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	//scope := r.FormValue("scope")
	reqUrl := fmt.Sprintf(`https://id.twitch.tv/oauth2/token?client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=https://burtbot.app/twitch_authcb`,
		os.Getenv("BB_APP_CLIENT_ID"),
		os.Getenv("BB_APP_SECRET"),
		code,
	)

	resp, err := http.Post(reqUrl, "text/html", strings.NewReader(""))
	if err != nil {
		log.Fatal("couldn't auth twitch token", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		bytes, _ := ioutil.ReadAll(resp.Body)
		log.Println(string(bytes))
		log.Fatal("couldn't communicate with twitch auth :, ", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	respObj := twitchAuthResp{}
	err = dec.Decode(&respObj)
	if err != nil {
		log.Fatal("couldn't parse twitch auth resp", err)
	}
	twitchAccessToken = respObj.Access_token
	twitchRefreshToken = respObj.Refresh_token
	twitchAuthCh <- true
	http.Redirect(w, r, "https://burtbot.app/services_auth", http.StatusSeeOther)
}

func GetAuthLink() string {
	var buf bytes.Buffer
	buf.WriteString("https://id.twitch.tv/oauth2/authorize")
	buf.WriteByte('?')
	v := url.Values{
		"client_id":     {os.Getenv("BB_APP_CLIENT_ID")},
		"redirect_uri":  {"https://burtbot.app/twitch_authcb"},
		"response_type": {"code"},
		"scope":         {"user:read:email"},
	}
	buf.WriteString(v.Encode())
	return buf.String()
}

func refreshAuth() {
	u := "https://id.twitch.tv/oauth2/token"
	v := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {twitchRefreshToken},
		"client_id":     {os.Getenv("BB_APP_CLIENT_ID")},
		"client_secret": {os.Getenv("BB_APP_SECRET")},
	}
	resp, err := http.PostForm(u, v)
	if err != nil {
		log.Println("Couldn't refresh twitch auth token", err)
		return
	}
	if resp.StatusCode == 400 {
		log.Println("Bad auth refresh request: ", resp.Status)
		return
	}
	dec := json.NewDecoder(resp.Body)
	r := twitchAuthResp{}
	dec.Decode(&r)
	twitchRefreshToken = r.Refresh_token
	twitchAccessToken = r.Access_token
}

func GetUser(username string) TwitchUser {
	u := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", username)
	req, err := http.NewRequest("GET", u, strings.NewReader(""))
	if err != nil {
		log.Println("Couldn't make request to twitch api", err)
		return TwitchUser{}
	}
	req.Header.Set("Authorization", "Bearer "+twitchAccessToken)
	req.Header.Set("Client-Id", os.Getenv("BB_APP_CLIENT_ID"))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Error making request to twitch api", err)
		return TwitchUser{}
	}
	if resp.StatusCode != 200 {
		log.Println("Bad request get user", resp.Status)
		return TwitchUser{}
	}

	dec := json.NewDecoder(resp.Body)
	rdata := struct {
		Data []TwitchUser
	}{}
	err = dec.Decode(&rdata)
	if err != nil {
		log.Println("Couldn't decode json from response", err)
		return TwitchUser{}
	}
	if len(rdata.Data) == 0 {
		return TwitchUser{}
	}
	return rdata.Data[0]
}

func GetChannelInfo(broadcaster_id string) ChannelInfo {
	u := fmt.Sprintf("https://api.twitch.tv/helix/channels?broadcaster_id=%s", broadcaster_id)
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Println("couldn't create request to get channel info: ", err)
		return ChannelInfo{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", twitchAppAccessToken))
	req.Header.Set("Client-Id", os.Getenv("BB_APP_CLIENT_ID"))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("couldn't make request to get channel info: ", err)
		return ChannelInfo{}
	}

	if resp.StatusCode != 200 {
		return ChannelInfo{}
	}

	respStruct := struct {
		Data []ChannelInfo
	}{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&respStruct)
	if err != nil {
		log.Println("couldn't decode json from channel info: ", err)
		return ChannelInfo{}
	}
	if len(respStruct.Data) == 0 {
		return ChannelInfo{}
	}
	return respStruct.Data[0]
}

func subscribe(event string) {
	u := "https://api.twitch.tv/helix/eventsub/subscriptions"
	cond := struct {
		BroadCasterUserID string `json:"broadcaster_user_id"`
	}{"38570305"}

	transport := struct {
		Method   string `json:"method"`
		Callback string `json:"callback"`
		Secret   string `json:"secret"`
	}{
		Method:   "webhook",
		Callback: os.Getenv("TWITCH_CALLBACK_URL"),
		Secret:   "supersecretsauce",
	}
	data := struct {
		Type      string `json:"type"`
		Version   string `json:"version"`
		Condition struct {
			BroadCasterUserID string `json:"broadcaster_user_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
			Secret   string `json:"secret"`
		} `json:"transport"`
	}{
		Type:      event,
		Version:   "1",
		Condition: cond,
		Transport: transport,
	}
	j, err := json.Marshal(data)
	if err != nil {
		log.Println("Could not marshal data to sub request: ", err)
		return
	}
	req, err := http.NewRequest("POST", u, bytes.NewReader(j))
	if err != nil {
		log.Println("Could not create request for sub", err)
		return
	}
	req.Header.Set("Client-ID", os.Getenv("BB_APP_CLIENT_ID"))
	req.Header.Set("Authorization", "Bearer "+twitchAppAccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("Could not make request to sub api: ", err)
		return
	}
	if resp.StatusCode != 200 {
		bs, _ := io.ReadAll(resp.Body)
		log.Println(string(bs))
		log.Println("Sub request: ", resp.StatusCode, resp.Status)
		return
	}
}

func EventSubCallback(w http.ResponseWriter, r *http.Request) {
	respStruct := struct {
		Challenge    string
		Subscription struct {
			ID        string
			Status    string
			Type      string
			Version   string
			Cost      int
			Condition struct {
				BroadCasterUserID string `json:"broadcaster_user_id"`
			}
			Transport struct {
				Method   string
				Callback string
			}
			CreatedAt time.Time `json:"created_at"`
		}
		Event struct {
			UserID               string `json:"user_id"`
			UserLogin            string `json:"user_login"`
			UserName             string `json:"user_name"`
			BroadcasterUserID    string `json:"broadcaster_user_id"`
			BroadcasterUserLogin string `json:"broadcaster_user_login"`
			BroadcasterUserName  string `json:"broadcaster_user_name"`
		}
	}{}
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("couldn't read bytes from request body: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	if ok := validSignature(r.Header, body); !ok {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	fmt.Println(r.Header.Get("Twitch-Eventsub-Message-Type"))
	err = json.Unmarshal(body, &respStruct)
	if err != nil {
		log.Println("Couldn't unmarshal response body: ", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	switch r.Header.Get("Twitch-Eventsub-Message-Type") {
	case "webhook_callback_verification":
		fmt.Fprintln(w, respStruct.Challenge)
	case "notification":
		if r.Header.Get("Twitch-Eventsub-Subscription-Type") == "channel.follow" && respStruct.Event.BroadcasterUserID == "38570305" {
			// notify subscribers of the follow event providing the username
			for _, f := range followEventSubscriptions {
				f(respStruct.Event.UserName)
			}
		}
		w.WriteHeader(200)
	}

}

func validSignature(headers http.Header, bodyBytes []byte) bool {
	hmacMsg := headers.Get("Twitch-Eventsub-Message-Id") + headers.Get("Twitch-Eventsub-Message-Timestamp") + string(bodyBytes)
	signature := hmac.New(sha256.New, []byte("supersecretsauce"))
	signature.Write([]byte(hmacMsg))
	exMAC := signature.Sum(nil)
	sentSig := strings.TrimPrefix(headers.Get("Twitch-Eventsub-Message-Signature"), "sha256=")
	sentMAC, err := hex.DecodeString(sentSig)
	if err != nil {
		log.Println("Error decoding request signature")
		return false
	}
	return hmac.Equal(exMAC, sentMAC)
}

func getAppAccessToken() string {
	u := "https://id.twitch.tv/oauth2/token"
	v := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {os.Getenv("BB_APP_CLIENT_ID")},
		"client_secret": {os.Getenv("BB_APP_SECRET")},
	}
	resp, err := http.PostForm(u, v)
	if err != nil {
		log.Println("Could not get app access token: ", err)
		return ""
	}
	respStruct := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&respStruct)
	if err != nil {
		log.Println("couldn't decode app access response: ", err)
		return ""
	}
	return respStruct.AccessToken
}

func deleteSubscription(id string) {
	u := fmt.Sprintf("https://api.twitch.tv/helix/eventsub/subscriptions?id=%s", id)
	req, err := http.NewRequest("DELETE", u, strings.NewReader(""))
	if err != nil {
		log.Println("Couldn't make request to cancel sub", err)
		return
	}
	req.Header.Set("Client-ID", os.Getenv("BB_APP_CLIENT_ID"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", twitchAppAccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("could not make request to cancel sub", err)
		return
	}
	fmt.Println("Cancel sub req status: ", resp.StatusCode)
}

func getSubscriptions() []EventSubscription {
	subData := struct {
		Data         []EventSubscription
		Total        int
		TotalCost    int `json:"total_cost"`
		MaxTotalCost int `json:"max_total_cost"`
		Limit        int
		Pagination   struct{}
	}{}
	u := "https://api.twitch.tv/helix/eventsub/subscriptions"
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		log.Println("couldn't create request to get subs: ", err)
		return []EventSubscription{}
	}
	req.Header.Set("Client-ID", os.Getenv("BB_APP_CLIENT_ID"))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", twitchAppAccessToken))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("couldn't make request to get subs: ", err)
		return []EventSubscription{}
	}

	if resp.StatusCode != 200 {
		log.Println(resp.Status, resp.StatusCode)
		return []EventSubscription{}
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&subData)
	if err != nil {
		log.Println("couldn't decode json getting subs: ", err)
		return []EventSubscription{}
	}
	return subData.Data
}

func GetAuthStatus() bool {
	return twitchAuth
}

func SubscribeToFollowEvent(fn func(string)) {
	followEventSubscriptions = append(followEventSubscriptions, fn)	
}
