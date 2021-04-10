package commands

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

type twitchAuthResp struct {
	Access_token  string
	Refresh_token string
	Expires_in    int
	Scope         []string
	Token_type    string
}

type TwitchUser struct {
	DisplayName   string `json:"display_name"`
	ProfileImgURL string `json:"profile_image_url"`
	ChannelDesc   string `json:"description"`
}

type TwitchAuthClient struct {
}

func (c *TwitchAuthClient) Init() {
	twitchAuthCh = make(chan bool)
	http.HandleFunc("/twitch_authcb", twitchAuthCb)
	http.HandleFunc("/twitch_link", getAuthLink)
	http.HandleFunc("/eventsub_cb", eventSubCallback)
	go http.ListenAndServe(":8078", nil)
	twitchAuth = <-twitchAuthCh
	fmt.Println("Auth'd for twitch api")
	twitchAppAccessToken = c.GetAppAccessToken()
	c.Subscribe("channel.follow")
}

func twitchAuthCb(w http.ResponseWriter, r *http.Request) {
	//fmt.Println(r.FormValue("code"))
	code := r.FormValue("code")
	//scope := r.FormValue("scope")
	reqUrl := fmt.Sprintf(`https://id.twitch.tv/oauth2/token?client_id=%s&client_secret=%s&code=%s&grant_type=authorization_code&redirect_uri=http://localhost:8078/twitch_authcb`,
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
		log.Fatal("couldn't communicate with twitch auth")
	}
	dec := json.NewDecoder(resp.Body)
	respObj := twitchAuthResp{}
	err = dec.Decode(&respObj)
	if err != nil {
		log.Fatal("couldn't parse twitch auth resp", err)
	}
	twitchAccessToken = respObj.Access_token
	twitchRefreshToken = respObj.Refresh_token
	fmt.Println("exp: ", respObj.Expires_in)
	fmt.Fprintf(w, "Twitch API authd!")
	twitchAuthCh <- true
}

func getAuthLink(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	buf.WriteString("https://id.twitch.tv/oauth2/authorize")
	buf.WriteByte('?')
	v := url.Values{
		"client_id":     {os.Getenv("BB_APP_CLIENT_ID")},
		"redirect_uri":  {"http://localhost:8078/twitch_authcb"},
		"response_type": {"code"},
		"scope":         {"user:read:email"},
	}
	buf.WriteString(v.Encode())
	fmt.Fprintf(w, `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><title>TwiAuth</title></head><body>Auth URL: <a href="%s">here</a></body></html>`, buf.String())
}

func (c *TwitchAuthClient) RefreshAuth() {
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

func (c *TwitchAuthClient) GetUser(username string) TwitchUser {
	u := fmt.Sprintf("https://api.twitch.tv/helix/users?login=%s", username)
	req, err := http.NewRequest("GET", u, strings.NewReader(""))
	if err != nil {
		log.Println("Couldn't make request to twitch api", err)
		return TwitchUser{}
	}
	req.Header.Set("Authorization", "Bearer "+twitchAccessToken)
	req.Header.Set("Client-Id", os.Getenv("BB_APP_CLIENT_ID"))
	resp, err := http.DefaultClient.Do(req)
	fmt.Println(resp.Request)
	if err != nil {
		log.Println("Error making request to twitch api", err)
		return TwitchUser{}
	}
	if resp.StatusCode != 200 {
		log.Println("Bad reqyest get user", resp.Status)
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
	return rdata.Data[0]
}

func (c *TwitchAuthClient) Subscribe(event string) {
	u := "https://api.twitch.tv/helix/eventsub/subscriptions"
	cond := struct {
		BroadCasterUserID string `json:"broadcaster_user_id"`
	}{"12826"}

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
		log.Println("Sub request: ", resp.StatusCode, resp.Status)
		return
	}
}

func eventSubCallback(w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintln(w, respStruct.Challenge)
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

func (c *TwitchAuthClient) GetAppAccessToken() string {
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
