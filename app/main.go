package main

import "bytes"
import "fmt"
import "github.com/gorilla/sessions"
import "github.com/labstack/echo-contrib/session"
import "github.com/labstack/echo/v4"
import "golang.org/x/oauth2"
import "golang.org/x/oauth2/google"
import "gopkg.in/yaml.v2"
import v2 "google.golang.org/api/oauth2/v2"
import "github.com/google/uuid"
import "io/ioutil"
import "net/http"
import "os"
import "text/template"
import "time"

func loadYaml() {

	buf, err := ioutil.ReadFile(".env.yml")
	if err != nil {
		panic(err)
	}

	m := make(map[interface{}]interface{})
	err = yaml.Unmarshal(buf, &m)
	if err != nil {
		panic(err)
	}
}

func configure(c echo.Context) *oauth2.Config {

	client_id := os.Getenv("CLIENT_ID")
	client_secret := os.Getenv("CLIENT_SECRET")

	conf := &oauth2.Config{
		ClientID: client_id,
		ClientSecret: client_secret,
		Scopes: []string{"email", "profile"},
		Endpoint: google.Endpoint,
		// Endpoint: oauth2.Endpoint{
		// 	AuthURL: "https://accounts.google.com/o/oauth2/v2/auth",
		// 	TokenURL: "https://www.googleapis.com/oauth2/v4/token",
		// },
		RedirectURL: "http://192.168.56.101.xip.io:8081/callback",
	}

	fmt.Printf("[TRACE]\n    oauth configuration: %v\n    endpoint: %v\n\n", conf, google.Endpoint)

	return conf
}

func onIndex(c echo.Context) error {

	fmt.Printf("[TRACE]\n    REQUEST: [/]\n\n")

	current_timestamp := time.Now().Format(time.RFC3339)
	id := ""
	email := ""

	// ========== セッション管理 ==========
	{
		sess, err := session.Get("session", c)
		if err != nil {
			c.Error(err)
			return err
		}
		fmt.Printf("[TRACE]\n    session: %v\n\n", sess.Values)
		if sess.Values["email"] != nil {
			fmt.Printf("[TRACE]\n    status: ログイン済み\n    id: %v\n    email: %v\n\n", sess.Values["id"], sess.Values["email"])
			id = sess.Values["id"].(string)
			email = sess.Values["email"].(string)
		} else {
			fmt.Printf("[TRACE]\n    status: 不明なユーザー\n    id: %v\n    email: %v\n\n", sess.Values["id"], sess.Values["email"])
		}
	}

	// ========== 応答 ==========
	{	
		t, _ := template.ParseFiles("templates/index.html")
		content := make(map[string]string)
		content["current_timestamp"] = current_timestamp
		content["id"] = id
		content["email"] = email
		buffer := new(bytes.Buffer)
		t.Execute(buffer, content)
		return c.HTML(http.StatusOK, string(buffer.Bytes()))
	}
}

func onTryOauthLoginCallback(c echo.Context) error {

	fmt.Printf("[TRACE]\n    REQUEST: [/callback]\n\n")

	// ========== Google から返却された情報を受け取っています ==========
	id := ""
	email := ""
	{
		conf := configure(c)
		code := c.QueryParam("code")
		context := oauth2.NoContext
		tok, err := conf.Exchange(context, code)
		if err != nil {
			fmt.Printf("[ERROR]\n    %v\n\n", err)
			return c.Redirect(http.StatusTemporaryRedirect, "http://192.168.56.101.xip.io:8081/")
		}
		if tok.Valid() == false {
			fmt.Printf("[ERROR]\n    invalid token.\n\n")
			return c.Redirect(http.StatusTemporaryRedirect, "http://192.168.56.101.xip.io:8081/")
		}
		service, _ := v2.New(conf.Client(context, tok))
		tokenInfo, _ := service.Tokeninfo().AccessToken(tok.AccessToken).Context(context).Do()

		fmt.Println("[TRACE]")
		fmt.Println("    status: ログイン成功！")
		fmt.Printf("    id: %v\n", tokenInfo.UserId)
		fmt.Printf("    email: %v\n", tokenInfo.Email)
		fmt.Println()

		id = tokenInfo.UserId
		email = tokenInfo.Email
	}

	// ========== セッション管理 ==========
	{
		fmt.Println("[TRACE]\n    セッションを初期化しています...\n\n")
		sess, err := session.Get("session", c)
		if err != nil {
			c.Error(err)
			return err
		}
		// sess.Options = &sessions.Options{ MaxAge: 86400 * 7, HttpOnly: true }
		sess.Values["id"] = id
		sess.Values["email"] = email
		sess.Save(c.Request(), c.Response())
	}

	fmt.Println("[TRACE]\n    ホームへリダイレクトしています...\n\n")
	return c.Redirect(http.StatusTemporaryRedirect, "http://192.168.56.101.xip.io:8081/")
}

func generateUUID() string {

	id := uuid.New()
	return fmt.Sprintf("%v", id)
}

func onTryOauthLogin(c echo.Context) error {

	fmt.Printf("[TRACE]\n    REQUEST: [/login]\n\n")

	conf := configure(c)
	randomString := generateUUID()
	url := conf.AuthCodeURL(randomString)
	return c.Redirect(http.StatusTemporaryRedirect, url)
}

func onTryOauthLogout(c echo.Context) error {

	fmt.Printf("[TRACE]\n    REQUEST: [/logout]\n\n")

	// ========== セッション管理 ==========
	{
		fmt.Println("[TRACE]\n    ログアウトしています...\n\n")
		sess, err := session.Get("session", c)
		if err != nil {
			c.Error(err)
			return err
		}
		sess.Values["id"] = nil
		sess.Values["email"] = nil
		sess.Save(c.Request(), c.Response())
	}

	return c.Redirect(http.StatusTemporaryRedirect, "http://192.168.56.101.xip.io:8081/")
}

func main() {

	e := echo.New()

	// ========== ミドルウェアを設定 ==========
	e.Use(session.Middleware(sessions.NewCookieStore([]byte("secret"))))

	// ========== ルーティングを定義 ==========
	e.GET("/", onIndex)
	e.GET("/login", onTryOauthLogin)
	e.GET("/logout", onTryOauthLogout)
	e.GET("/callback", onTryOauthLoginCallback)

	// ========== サーバーを実行 ==========
	e.Logger.Fatal(e.Start(":8081"))
}
