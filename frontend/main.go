package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

var pages *template.Template

type analysisFindings struct {
	Severity string `json:"severity"`
	File     string `json:"file"`
	Text     string `json:"text"`
}

type analysisItem struct {
	Catagory         string             `json:"catagory"`
	Name             string             `json:"name"`
	Language         string             `json:"language"`
	AnalysisFindings []analysisFindings `json:"findings"`
}

type analysisReport struct {
	Hash         string         `json:"hash"`
	Log          string         `json:"log"`
	Error        string         `json:"error"`
	AnalysisItem []analysisItem `json:"analysis"`
}

/*

{
	hash: "112233445566778899aa...",
	log: "[INFO] .... \n [ERROR] junk....",
	error: false,
	analysis: [
	  {
		 category: "linting"
		 name: "binary imports (objdump)",
		 language: "binary",
		 findings: [
	   {
			 severity: "warning",
			 file: "./server",
			 text: "import of gets(2) detected"
			}
			...
		 ]
	  }
	  ...
	]
  }
*/
func init() {
	var err error
	pages, err = template.ParseGlob("./web/templates/*")
	if err != nil {
		panic(err)
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func upload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "POST" {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			fmt.Println(err)
			return
		}
		f, err := os.Create("./uploads/" + handler.Filename)

		buf := bytes.NewBuffer(nil)
		io.Copy(buf, file)
		f.Write(buf.Bytes())
		defer f.Close()
		defer file.Close()

		h := sha256.New()

		h.Write(buf.Bytes())

		bs := h.Sum(nil)

		f1, _ := os.Create("./analysis/" + hex.EncodeToString(bs))
		f2, _ := os.Create("./binaries/" + hex.EncodeToString(bs))

		defer f1.Close()
		defer f2.Close()

		f2.Write(buf.Bytes())

		http.Redirect(w, r, "/reports/"+hex.EncodeToString(bs), 301)
	}
}

func getReport(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "GET" {
		fmt.Println(r.RequestURI)
		hash := strings.Split(r.RequestURI, "/")[2]

		fa, err := os.Stat("./analysis/" + hash)

		if os.IsNotExist(err) || fa.Size() == 0 {
			// Analysis does not exist or file is empty
			fmt.Println("Accessed in progress analysis" + hash)
			err = pages.ExecuteTemplate(w, "waiting.html", nil)
			fmt.Println(err)

		} else {
			// Analysis is complete
			fmt.Println("Accessed completed analysis" + hash)

			frep, ferr := ioutil.ReadFile("./analysis/" + hash)
			check(ferr)
			rawreport := frep
			jsonreport := analysisReport{}
			json.Unmarshal([]byte(rawreport), &jsonreport)

			fmt.Println(jsonreport)

			err = pages.ExecuteTemplate(w, "report.html", jsonreport)
			fmt.Println(err)

		}

		//p, _ := loadPage(title)
		//fmt.Fprintf(w, "<h1>%s</h1><div>%s</div>", p.Title, p.Body)

		fmt.Println(hash)
	}
}

func main() {
	fmt.Println("Starting go web server on port 8080")
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/reports/", getReport)
	http.Handle("/", http.FileServer(http.Dir("./web")))
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}