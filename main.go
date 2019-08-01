package main

import (
	"context"
	"fmt"
	"github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main(){
	if len(os.Args) < 2{
		fmt.Fprintf(os.Stderr, "Not Enough Args\n")
		os.Exit(1)
	}
	box := packr.New("assets", "./assets")
	indexPage, err := box.Find("index.html")
	fatal(err)
	respRaw, err := box.FindString("response.html")
	fatal(err)
	respTmpl, err := template.New("result").Parse(respRaw)
	fatal(err)
	progRun := handlerMaker(os.Args[1], os.Args[2:])
	mux := mux.NewRouter()
	// Handler Plain Text Post
	mux.HandleFunc("/",func(resp http.ResponseWriter, req *http.Request){
		output, err := progRun(req.Body)
		if err != nil{
			log.Println(err)
		}
		resp.Header().Add("Content-Type", "text/plain; charset=utf-8")
		resp.Write([]byte(output))
	}).Methods("POST").Headers("Content-Type","text/plain")
	// Handle MultipartForm
	mux.HandleFunc("/",func(resp http.ResponseWriter, req *http.Request){
		multipart, err := req.MultipartReader()
		var builder strings.Builder
		fatal(err)
		for{
			part, err := multipart.NextPart();
			if err != nil{
				if err != io.EOF{
					log.Println(err)
				}
				break
			}
			output, err := progRun(part)
			if err != nil{
				log.Println(err)
			}
			builder.WriteString(output)
		}
		respTmpl.Execute(resp, builder.String())
		resp.Header().Add("Content-Type", "text/plain; charset=utf-8")
	}).Methods("POST").HeadersRegexp("Content-Type","multipart/form-data*")
	// Handle Static index
	mux.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request){
		resp.Write(indexPage)
	}).Methods("GET")
	log.Fatal(http.ListenAndServe(":8787", mux))
}

func handlerMaker(prog string, arg []string)func(io.ReadCloser)(string,error){
	return func(in io.ReadCloser)(string, error){
		contx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		cmd := exec.CommandContext(contx, prog, arg...)
		stdin, err := cmd.StdinPipe()
		if err != nil{
			return "", err
		}
		go func(){
			defer stdin.Close()
			_, err := io.Copy(stdin, in)
			if err != nil{
				log.Println(err)
			}
		}()
		raw, err := cmd.Output()
		in.Close()
		if err != nil{
			return "", err
		}
		return string(raw), nil
	}
}

func fatal(err error){
	if err != nil{
		log.Fatalln(err)
	}
}