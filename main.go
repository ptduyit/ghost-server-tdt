package main

import (
	"bytes"
	"encoding/base64"

	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/getlantern/systray"
	"github.com/gorilla/websocket"
	"github.com/kataras/iris/v12"
	"github.com/nfnt/resize"
)

type AnswerClient struct {
	Md5    string `json:"md5"`
	Answer string `json:"answer"`
}

type CaptchaInfo struct {
	Md5        string
	AnswerList string
	Answer     string
	Base64     string
}

var clients = make(map[*websocket.Conn]bool) // connected clients
var broadcast = make(chan AnswerClient)      // broadcast channel

// Configure the upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var mapCaptcha = make(map[string]Captcha)
var mapAnswer = make(map[string]string)
var mapCaptchaImage = make(map[string]Captcha)

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go runSystray(&wg)
	go runApi(&wg)
	wg.Wait()
}

type Config struct {
	Login    string
	UserInfo string
	Mission  string
	Map      string
}

type Captcha struct {
	Answers string
	Md5     string
	image   image.Image
	Base64  string
}

func runApi(wg *sync.WaitGroup) {
	var conf Config
	if _, err := toml.DecodeFile("settings.toml", &conf); err != nil {
		// handle error
		println("can't load file settings", err)
	}

	// RestoreAssets("","static")
	app := iris.Default()
	// server tieudattai - tanthienlong3d

	app.Post("/microauto/user.php", func(ctx iris.Context) {
		act := ctx.FormValue("cmd")
		if act == "login" {
			ctx.Writef(conf.Login)
		} else if act == "msg" {
			version := ctx.FormValue("version")
			message := checkUpdate(version)
			ctx.Writef(message)
		} else if act == "reload" {
			ctx.Writef(`pro`)
		} else if act == "userinfo" {
			ctx.Writef(conf.UserInfo)
		} else if act == "ini" {
			ctx.Writef(`Save success`)
		}
	})

	app.Get("/microauto/user.php", func(ctx iris.Context) {
		act := ctx.FormValue("cmd")
		if act == "getplayerinfo" || act == "ini" {
			ctx.Writef(``)
		}
	})

	app.Get("/Ngoc.php", func(ctx iris.Context) {
		ctx.Writef(
			`50202003	Hồng Tinh Thạch (Cấp 2)
50202002	Lam Tinh Thạch (Cấp 2)
50202004	Lục Tinh Thạch (Cấp 2)
50202001	Hoàng Tinh Thạch (Cấp 2)
50213004	Hồng Bảo Thạch (Cấp 2)
50201001	Miêu Nhãn Thạch (Cấp 2)
50102003	Hồng Tinh Thạch (Cấp 1)
50102002	Lam Tinh Thạch (Cấp 1)
50102004	Lục Tinh Thạch (Cấp 1)
50102001	Hoàng Tinh Thạch (Cấp 1)`)
	})

	app.Get("/Help.php", func(ctx iris.Context) {
		ctx.Writef(``)
	})

	app.Get("/ad.php", func(ctx iris.Context) {
		ctx.Writef(``)
	})

	app.Get("/map.php", func(ctx iris.Context) {
		ctx.Writef(conf.Map)
	})

	app.Post("/microauto/captcha.php", func(ctx iris.Context) {
		cmd := ctx.FormValue("cmd")
		if cmd == "pushcap" {
			// ctx.Writef(``)
			// return
			postform := ctx.Request().PostForm
			bodyjson, _ := json.Marshal(postform)
			bodystring := string(bodyjson)
			substring := bodystring[2 : len(bodystring)-7]
			splitString := strings.Split(substring, "|")
			md5code := splitString[0]
			bitmapAnswer := splitString[1]
			bitmapcode := bitmapAnswer[0 : len(bitmapAnswer)-16]
			answers := bitmapAnswer[len(bitmapAnswer)-16:]

			upLeft := image.Point{0, 0}
			lowRight := image.Point{128, 36}

			img := image.NewRGBA(image.Rectangle{upLeft, lowRight})
			var b strings.Builder
			for x := 0; x < len(bitmapcode); x++ {
				b.WriteString(hexToBin(string(bitmapcode[x])))
			}
			binstring := b.String()
			num := 0
			for x := 0; x < len(binstring); x++ {
				if binstring[x] != '0' {
					img.Set(num/2%128, num/2/128, color.White)
				} else {
					img.Set(num/2%128, num/2/128, color.Black)
				}
				num += 2
			}

			m := resize.Resize(160, 45, img, resize.Lanczos3)

			buf := new(bytes.Buffer)
			err := jpeg.Encode(buf, m, nil)
			if err != nil {
				fmt.Println("failed to create buffer", err)
			}

			// if _, err := os.Stat("captcha-test/" + answers + ".jpg"); errors.Is(err, os.ErrNotExist) {
			// 	f, err := os.Create("captcha-test/" + answers + ".jpg")
			// 	if err != nil {
			// 		panic(err)
			// 	}
			// 	defer f.Close()
			// 	if err = jpeg.Encode(f, m, nil); err != nil {
			// 		log.Printf("failed to encode: %v", err)
			// 	}
			// }
			// f, err := os.OpenFile("suggest.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
			// if err != nil {
			// 	panic(err)
			// }

			// defer f.Close()

			// if _, err = f.WriteString(answers + " "); err != nil {
			// 	panic(err)
			// }

			bufByte := buf.Bytes()
			imgBase64Str := base64.StdEncoding.EncodeToString(bufByte)
			// println(imgBase64Str)
			id := bypasscaptcha(imgBase64Str)
			captcha := Captcha{Md5: md5code, Answers: answers, image: m, Base64: imgBase64Str}
			// mapCaptcha[id] = append(mapCaptcha[id].Md5, md5code)
			// mapCaptcha[id] = append(mapCaptcha[id], answers)
			// mapCaptcha[id] = append(mapCaptcha[id], imgBase64Str)
			mapCaptcha[id] = captcha
			fmt.Println("request captcha id: " + id + " with answer: " + answers)
			ctx.Writef(``)
		}

		if cmd == "popex" {
			// time.Sleep(60 * time.Second)
			// ctx.Writef(``)
			// return
			var textCap strings.Builder

			stop := 0
			for len(mapAnswer) == 0 {
				time.Sleep(3 * time.Second)
				stop += 3
				if stop > 25 {
					ctx.Writef(``)
					return
				}
			}

			bodyCaptcha := ""
			for key, value := range mapAnswer {
				textCap.WriteString(key)
				textCap.WriteString(":")
				textCap.WriteString(value)
				textCap.WriteString("|")
				captchaMap := mapCaptchaImage[key]
				writeFileImage(value, captchaMap.image)
				delete(mapAnswer, key)
				delete(mapCaptchaImage, key)
			}

			bodyCaptcha = textCap.String()
			bodyCaptcha = strings.TrimSuffix(bodyCaptcha, "|")

			fmt.Println("Response Client", bodyCaptcha)
			ctx.Writef(bodyCaptcha)
		}

	})
	app.Get("/microauto/ScriptExs.php", func(ctx iris.Context) {
		ctx.Writef(conf.Mission)
	})

	// Create a simple file server
	fs := http.FileServer(http.Dir("./public"))
	http.Handle("/", fs)

	// Configure websocket route
	http.HandleFunc("/ws", handleConnections)

	go handleMessages()
	go loopRequestCaptchaResolve()

	// Start the server on localhost port 8000 and log any errors
	go http.ListenAndServe(":8000", nil)

	go app.Listen(":80")
	app.Run(iris.TLS(":1997", "static\\service.crt", "static\\service.key"))
	wg.Done()
	os.Exit(2)
}

func loopRequestCaptchaResolve() {
	for {
		time.Sleep(2 * time.Second)
		for key, value := range mapCaptcha {
			if checkDuplicateText(value.Answers) {
				fmt.Println("captcha wrong duplicate character", value.Answers)
				delete(mapCaptcha, key)
			} else {
				response := getCaptchaValue(key)
				if response == "UNSOLVABLE" {
					delete(mapCaptcha, key)
					fmt.Println("anwser from api UNSOLVABLE or Wrong ID")
				} else if response != "" {
					fmt.Println("anwser from api ", response)
					answerCorrect := checkcaptchaCorrect(response, value.Answers)
					if answerCorrect != "" {
						// writeFileImage(answerCorrect, value.image)
						// mapAnswer[value.Md5] = answerCorrect
						mapCaptchaImage[value.Md5] = value
						for client := range clients {
							err := client.WriteJSON(CaptchaInfo{Md5: value.Md5, AnswerList: value.Answers, Answer: answerCorrect, Base64: value.Base64})
							if err != nil {
								log.Printf("error: %v", err)
								client.Close()
								delete(clients, client)
							}
						}
					}
					delete(mapCaptcha, key)
				}
			}
		}
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade initial GET request to a websocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	// Make sure we close the connection when the function returns
	defer ws.Close()

	// Register our new client
	clients[ws] = true

	for {
		var msg AnswerClient
		// Read in a new message as JSON and map it to a Message object
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, ws)
			break
		}
		// Send the newly received message to the broadcast channel
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		// Grab the next message from the broadcast channel
		msg := <-broadcast
		// Send it out to every client that is currently connected
		mapAnswer[msg.Md5] = msg.Answer
	}
}

func checkDuplicateText(text string) bool {
	for _, c := range text {
		count := strings.Count(text, string(c))
		if count > 1 {
			return true
		}
	}
	return false
}

func runSystray(wg *sync.WaitGroup) {
	onReady := func() {
		icon, error := ioutil.ReadFile("icon.ico")
		if error != nil {
			fmt.Println("err =", error)
		}
		systray.SetIcon(icon)
		systray.SetTitle("Server tieudattai")
		systray.SetTooltip("Server tieudattai")
		mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

		go func() {
			<-mQuit.ClickedCh
			systray.Quit()
		}()
	}

	onExit := func() {
		wg.Done()
		os.Exit(1)
	}
	systray.Run(onReady, onExit)
}

func checkUpdate(version string) string {
	apiUrl := "https://tieudattai.org"
	resource := "/microauto/user.php"
	data := url.Values{}
	data.Set("cmd", "msg")
	data.Set("version", version)

	u, _ := url.ParseRequestURI(apiUrl)
	u.Path = resource
	urlStr := u.String()

	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode())) // URL-encoded payload
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := client.Do(r)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return ""
		}
		bodyString := string(bodyBytes)
		return bodyString
	}
	return ""
}

func bypasscaptcha(body string) string {
	apiUrl := "http://127.0.0.1:80"
	resource := "/in.php"
	data := url.Values{}
	data.Set("method", "base64")
	// data.Set("min_len", "1")
	// data.Set("max_len", "2")
	// data.Set("submib", "download+and+get+the+ID")
	data.Set("body", body)

	u, _ := url.ParseRequestURI(apiUrl)
	u.Path = resource
	urlStr := u.String()

	client := &http.Client{}
	r, _ := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode())) // URL-encoded payload
	r.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := client.Do(r)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return ""
		}
		bodyString := string(bodyBytes)
		trim := strings.TrimSpace(bodyString)
		fmt.Println(trim)
		id := strings.Split(trim, "|")
		return id[1]
	}
	return ""
}

func getCaptchaValue(id string) string {
	apiUrl := "http://127.0.0.1:80"
	resource := "/res.php"

	u, _ := url.ParseRequestURI(apiUrl)
	u.Path = resource
	urlStr := u.String()

	client := &http.Client{}
	r, err := http.NewRequest(http.MethodPost, urlStr, nil) // URL-encoded payload
	if err != nil {
		log.Print(err)
	}

	q := r.URL.Query()
	q.Add("action", "get")
	q.Add("id", id)
	r.URL.RawQuery = q.Encode()

	resp, _ := client.Do(r)
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return ""
		}
		bodyString := string(bodyBytes)
		if strings.Contains(bodyString, "|") {
			body := strings.Split(bodyString, "|")
			return body[1]
		} else if strings.Contains(bodyString, "UNSOLVABLE") || strings.Contains(bodyString, "WRONG_CAPTCHA_ID") {
			return "UNSOLVABLE"
		}
	}
	return ""
}

func hexToBin(hex string) string {
	switch hex {
	case "0":
		return "0000"
	case "1":
		return "0001"
	case "2":
		return "0010"
	case "3":
		return "0011"
	case "4":
		return "0100"
	case "5":
		return "0101"
	case "6":
		return "0110"
	case "7":
		return "0111"
	case "8":
		return "1000"
	case "9":
		return "1001"
	case "A":
		return "1010"
	case "B":
		return "1011"
	case "C":
		return "1100"
	case "D":
		return "1101"
	case "E":
		return "1110"
	case "F":
		return "1111"
	}
	return ""

}

func checkcaptchaCorrect(textCheck string, answers string) string {
	var arrAnswers [4]string
	arrAnswers[0] = answers[:4]
	arrAnswers[1] = answers[4:8]
	arrAnswers[2] = answers[8:12]
	arrAnswers[3] = answers[12:]

	textCheck = strings.ToUpper(textCheck)
	// fmt.Println("answer server: ", textCheck)
	// textCheck = appendCharacter(textCheck)
	// fmt.Println("answer appendCharacter: ", textCheck)
	// fmt.Println("full answer: ", answers)
	max := 0
	index := 0
	isWrong := true

	for i := 0; i < 4; i++ {
		duplicate := 0
		for x := 0; x < len(textCheck); x++ {
			if strings.Contains(string(arrAnswers[i]), string(textCheck[x])) {
				duplicate++
				isWrong = false
			}
		}
		if duplicate > max {
			max = duplicate
			index = i
		}
	}
	if isWrong {
		fmt.Println("captcha from api not match answer: " + answers + " - " + textCheck)
		// return ""
	}
	return arrAnswers[index]
}

func writeFileImage(name string, image image.Image) {
	if _, err := os.Stat("captcha/" + name + ".jpg"); errors.Is(err, os.ErrNotExist) {
		f, err := os.Create("captcha/" + name + ".jpg")
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err = jpeg.Encode(f, image, nil); err != nil {
			log.Printf("failed to encode: %v", err)
		}
	}
}

func appendCharacter(text string) string {
	var textCap strings.Builder
	textDuplicate := removeDuplicate(text)
	textCap.WriteString(textDuplicate)
	for i := 0; i < len(textDuplicate); i++ {
		char := string(textDuplicate[i])
		if char == "0" {
			textCap.WriteString("OQD")
		}
		if char == "1" {
			textCap.WriteString("7IJ")
		}
		if char == "3" {
			textCap.WriteString("B")
		}
		if char == "4" {
			textCap.WriteString("A")
		}
		if char == "5" {
			textCap.WriteString("S")
		}
		if char == "6" {
			textCap.WriteString("CGO0Q")
		}
		if char == "7" {
			textCap.WriteString("1IJ")
		}
		if char == "9" || char == "D" {
			textCap.WriteString("OQ")
		}
		if char == "B" {
			textCap.WriteString("3")
		}
		if char == "C" {
			textCap.WriteString("GSO")
		}
		if char == "K" {
			textCap.WriteString("R")
		}
		if char == "M" {
			textCap.WriteString("N")
		}
		if char == "N" {
			textCap.WriteString("M")
		}
		if char == "O" {
			textCap.WriteString("06Q")
		}
		if char == "Q" {
			textCap.WriteString("O09")
		}
		if char == "S" {
			textCap.WriteString("5")
		}
		if char == "V" {
			textCap.WriteString("U")
		}
		if char == "W" {
			textCap.WriteString("MV")
		}

	}
	return removeDuplicate(textCap.String())
}

func removeDuplicate(stringSlice string) string {
	keys := make(map[string]bool)
	var list strings.Builder
	for _, entry := range stringSlice {
		entryString := string(entry)
		if _, value := keys[entryString]; !value {
			keys[entryString] = true
			list.WriteString(entryString)
		}
	}
	return list.String()
}
