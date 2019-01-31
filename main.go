package main

import (
	"flag"
	"net/http"
	"strings"
	"net/url"
	"net/http/httputil"
	"sync"
	"io/ioutil"
	"fmt"
	"time"
	"log"
	"compress/gzip"
	"regexp"
)


var upstream  = ""
var listen  = "8899"
var pattern = ""
var responeHeader = ""
var responseBody = ""
var responseCode int
var silence bool

func main()  {


	flag.StringVar(&upstream,"u","","被代理的服务器 eg: 172.30.0.100:8080")
	//flag.StringVar(&upstream,"u","172.20.1.120:8080","upstream (real server) no default value")
	flag.StringVar(&listen,"l","8899","监听的端口")
	flag.StringVar(&pattern,"p",".*","过滤URL 支持正则表达式")

	flag.StringVar(&responeHeader,"h","Content-Type: application/json;charset=utf-8",
		"直接响应 HEADE \r\n例如： -h \"Connection: keep-alive,Content-Type: application/json\"\r\n")
	flag.StringVar(&responseBody,"b","default body","直接相应 BODY， 可以直接跟字符串，亦可接文件（@filepath） \r\n" +
		"例如: -b \"{\\\"name\\\":\\\"value\\\"}\"\r\n" +
		"      -b @file \r\n")
	flag.BoolVar(&silence,"s",false,"不打印请求响应内容")
	flag.IntVar(&responseCode,"c",200,"直接响应 code (默认200)")
	flag.Parse()


	log.Print("listen: ", listen, " upstream : ", upstream)


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		index := getIndex()


		log.Print("[",index,"]",r.Method,r.URL.Path)



		if strings.Contains(r.URL.Path,"favicon"){
			return
		}

		if upstream == "" {
			debugRequest(index,r)

			for k,v := range parseCmdHeader(responeHeader){
				w.Header().Set(k,v)
			}
			w.WriteHeader(responseCode)

			bd,err := parseCmdBody(responseBody)
			if err != nil{
				log.Fatalln("parse body failed",err)
			}

			w.Write([]byte(bd))

			return
		}
		reg := regexp.MustCompile(pattern)
		if reg.MatchString(r.URL.Path) && !silence{
			debugRequest(index,r)
		}



		director := func(req *http.Request) {

			target,_ := url.Parse("http://" + upstream)
			targetQuery := target.RawQuery
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = singleJoiningSlash(target.Path, req.URL.Path)
			if targetQuery == "" || req.URL.RawQuery == "" {
				req.URL.RawQuery = targetQuery + req.URL.RawQuery
			} else {
				req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				// explicitly disable User-Agent so it's not set to default value
				req.Header.Set("User-Agent", "")
			}

		}
		modifyResp :=  func(resp *http.Response) error{
			if reg.MatchString(r.URL.Path) && !silence{
				debugResponse(index,resp)
			}



			return nil
		}
		proxy := &httputil.ReverseProxy{Director:director, ModifyResponse:modifyResp}

		proxy.ServeHTTP(w,r)

	})

	log.Fatalln(http.ListenAndServe(":"+listen, nil))

}


var index = 0
func getIndex() string  {

	mux := sync.Mutex{}
	mux.Lock()
	defer mux.Unlock()

	result := fmt.Sprintf("%v_%v",time.Now().UnixNano(),index)
	index++
	return result
}

func debugResponse(index string ,resp *http.Response)  {

	log.Print("[",index,"]Response <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	log.Print("[",index,"]Header ------------------------- ")
	log.Print("[",index,"] ",resp.Proto," ",resp.Status)
	for k,v := range resp.Header{
		for _,v2 := range  v{
			log.Print("[",index,"] ",k," : ",v2)
		}
	}
	log.Print("[",index,"]Body --------------------------")
	zip := resp.Header.Get("Content-Encoding")
	var bt []byte
	switch zip {
	case "gzip":{
		reader,err := gzip.NewReader(resp.Body)
		if err != nil{
			log.Fatalln(err)
		}
		bt,_ = ioutil.ReadAll(reader)
		body := string(bt)
		log.Print("[",index,"]", body)
		resp.Body= ioutil.NopCloser(strings.NewReader(body))
		resp.ContentLength = int64(len(body))
		resp.Header.Set("Content-Length",fmt.Sprintf("%v",resp.ContentLength))
		resp.Header.Del("Content-Encoding")

	}
	default:
		bt,_ = ioutil.ReadAll(resp.Body)
		body := string(bt)
		log.Print("[",index,"]", body)
		resp.Body= ioutil.NopCloser(strings.NewReader(body))

	}



	log.Print("[",index,"]ResponseEnd <<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
}

func debugRequest(index string, req *http.Request)  {
	log.Print("[",index,"]Request >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	log.Print("[",index,"]Header ------------------------- ")
	log.Print("[",index,"] ",req.Method," ",decodeurl(req.URL.Path)," ",req.Proto)
	log.Print("[",index,"] Host:",req.Host)
	for k,v := range req.Header{
		for _,v2 := range  v{
			log.Print("[",index,"] ",k," : ",v2)
		}
	}
	log.Print("[",index,"]Body --------------------------")

	bt,_ := ioutil.ReadAll(req.Body)

	body := string(bt)
	log.Print("[",index,"]",body)
	req.Body= ioutil.NopCloser(strings.NewReader(body))
	log.Print("[",index,"]RequestEnd >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
}


func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	}
	return a + b
}

func decodeurl(path string) string  {
	rs,err := url.PathUnescape(path)
	if(err != nil){
		return path
	}else{
		return rs
	}
}

func parseCmdHeader(str string) map[string]string {
	var m = make(map[string]string)
	if str == ""{
		return m
	}else{
		items := strings.Split(str,",")
		for _,item := range items{
			kv :=strings.Split(item,":")
			if len(kv) == 2{
				k := strings.TrimSpace(kv[0])
				v := strings.TrimSpace(kv[1])
				m[k] = v
			}else{
				continue
			}
		}
	}
	return m
}

func parseCmdBody(str string) (string,error)  {
	if strings.HasPrefix(str,"@"){
		path := strings.TrimLeft(str,"@")
		bt,err := ioutil.ReadFile(path)
		if err != nil{
			return "",err
		}else{
			return string(bt),nil
		}
	}else{
		return  str,nil
	}
}