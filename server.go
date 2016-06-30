package main

import (
	"fmt"
	"github.com/kr/pty"
	"github.com/pkg/browser"
	"golang.org/x/net/html"
	"golang.org/x/net/websocket"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var loggerF *log.Logger
var logFile *os.File = os.Stdout

type tty struct {
	cmdprs  *exec.Cmd
	running bool
}

func init() {
	os.Mkdir("genCodes", 0777)
	os.Chmod("genCodes", 0777)
	logFile, err := os.Create("log.txt")
	if err != nil {
		fmt.Println("ERR:", err)
	} else {
		loggerF = log.New(logFile, "goEngage", log.LstdFlags)
	}
}

func runHandler(ws *websocket.Conn) {
	fmt.Println("runHandler open")
	type msg struct {
		Category string
		Data     string
	}
	process := tty{}
	var cmdtty *os.File
	var timeStamp = time.Now()

	for {
		var wsInput msg
		err := websocket.JSON.Receive(ws, &wsInput)
		if err != nil {
			loggerF.Println("ERR ", err)
			break
		}
		var programfile string
		switch wsInput.Category {
		case "getLink":
			resp, err := http.Get(wsInput.Data)
			if err != nil {
				fmt.Println(err)
			}
			defer resp.Body.Close()
			z := html.NewTokenizer(resp.Body)
			foundProgamCode := false
			var programcode html.Token
			for {
				tt := z.Next()
				switch {
				case tt == html.ErrorToken:
					// End of the document, we're done
					return
				case tt == html.StartTagToken:
					t := z.Token()
					isAnchor := t.Data == "textarea"
					if isAnchor {
						z.Next()
						programcode = z.Token()
						// fmt.Println(programcode.Data)
						foundProgamCode = true
					}
				}
				if foundProgamCode {
					break
				}
			}
			var output msg
			output.Category = "getLink"
			output.Data = fmt.Sprint(programcode.Data)
			err = websocket.JSON.Send(ws, output)
			if err != nil {
				fmt.Println("err :", err)
			}
		case "share":
			resp, err := http.Post("https://play.golang.org/share",
				"application/x-www-form-urlencoded", strings.NewReader(wsInput.Data))
			if err != nil {
				fmt.Println(err)
			}
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("ERR :", err)
			}
			defer resp.Body.Close()
			var output msg
			output.Category = "share"
			output.Data = fmt.Sprintf("http://play.golang.org/p/%v", string(body))
			err = websocket.JSON.Send(ws, output)
			if err != nil {
				fmt.Println("err :", err)
			}
		case "format":
			programfile = createFile(wsInput.Data, timeStamp)
			goFmt := exec.Command("gofmt", "-w", programfile)
			goFmt.Run()
			fileFormated, err := ioutil.ReadFile(programfile)
			if err != nil {
				loggerF.Println("ERR :", err)
			} else {
				var output msg
				output.Category = "format"
				output.Data = string(fileFormated)
				websocket.JSON.Send(ws, output)
			}
		case "code":
			programfile = createFile(wsInput.Data, timeStamp)
			cmd := exec.Command("go", "run", programfile)
			process.cmdprs = cmd
			process.running = true
			cmdtty, err = pty.Start(process.cmdprs)
			go io.Copy(ws, cmdtty)
		case "input":
			if process.running {
				// fmt.Printf("wrtting \"%v\" to STDIN\n",wsInput.Data)
				cmdtty.WriteString(wsInput.Data + "\n")
			}
		case "stop":
			if process.running {
				process.cmdprs.Process.Kill()
			}
		default:
			fmt.Println("UNKNOWN wsInput.Category")

		}

	}
	fmt.Println("runHandler Close")

}
func createFile(content string, timeStamp time.Time) (filename string) {
	filename = fmt.Sprint("genCodes", string(os.PathSeparator), timeStamp.Format(time.RFC1123), ".go")
	file, err := os.Create(filename)
	if err != nil {
		loggerF.Println(err, "os.Create")
	}
	_, err = file.WriteString(content)
	if err != nil {
		loggerF.Println(err, "WriteString(hw)")
	}
	return filename
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	requsetedFile := strings.Join([]string{"static", r.URL.Path[1:]}, string(os.PathSeparator))
	http.ServeFile(w, r, requsetedFile)
}

func main() {
	fmt.Println("Starting GoEngage")
	http.HandleFunc("/", staticHandler)
	http.Handle("/run", websocket.Handler(runHandler))
	serverport := ":" + os.Getenv("PORT")
	if serverport == ":" {
		serverport = ":8080"
	}
	fmt.Println("starting server.go \n Listening @ localhost", serverport)

	err := browser.OpenURL(fmt.Sprintf("http://localhost%v", serverport))
	if err != nil {
		panic("Error: " + err.Error())
	}

	err = http.ListenAndServe(serverport, nil)
	if err != nil {
		panic("Error: " + err.Error())
	}

}
