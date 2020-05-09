package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/pkg/errors"
	cipher "github.com/xxzl0130/GF_Kalina/GF_cipher"
	"github.com/xxzl0130/GF_Kalina/pkg/util"
)

func main() {
	gf := &GF{
		ch:   make(chan response, 128),
		key : "",
	}
	if err := gf.Run(); err != nil {
		fmt.Printf("程序启动失败 -> %+v\n", err)
	}
}

type response struct {
	Host string
	Path string
	Body []byte
}

type GF struct {
	ch   chan response
	key string
}

func (gf *GF) Run() error {
	go gf.loop()

	localhost, err := gf.getLocalhost()
	if err != nil {
		fmt.Printf("获取代理地址失败 -> %+v\n", err)
		return err
	}

	fmt.Printf("代理地址 -> %v:%v\n", localhost, 8888)

	srv := goproxy.NewProxyHttpServer()
	srv.Logger = new(util.NilLogger)
	//srv.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	srv.OnResponse(gf.condition()).DoFunc(gf.onResponse)

	if err := http.ListenAndServe(":8888", srv); err != nil {
		fmt.Printf("启动代理服务器失败 -> %+v\n", err)
		return err
	}

	return nil
}

func (gf *GF) build(body response) {
	type Uid struct {
		Sign string `json:"sign"`
	}
	type KalinaData struct {
		Level string `json:"level"`
		Favor string `json:"favor"`
	}
	type GF_Json struct {
		User map[string]interface{} `json:"user_record"`
		Kalina KalinaData `json:"kalina_with_user_info"`
	}
	
	// starts with "#"
	if body.Body[0] == byte(35){
		if strings.HasSuffix(body.Path,"/Index/getDigitalSkyNbUid") || strings.HasSuffix(body.Path, "/Index/getUidTianxiaQueue") || strings.HasSuffix(body.Path,"/Index/getUidEnMicaQueue"){
			data, err := cipher.AuthCodeDecodeB64Default(string(body.Body)[1:])
			if err != nil {
				fmt.Printf("解析Uid数据失败 -> %+v\n", err)
				return
			}
			uid := Uid{}
			if err := json.Unmarshal([]byte(data), &uid); err != nil {
				fmt.Printf("解析JSON数据失败 -> %+v\n", err)
				return
			}
			gf.key = uid.Sign
			return
		} else if strings.HasSuffix(body.Path,"/Index/index"){
			data, err := cipher.AuthCodeDecodeB64(string(body.Body)[1:], gf.key, true)
			if err != nil {
				fmt.Printf("解析用户数据失败 -> %+v\n", err)
				return
			}
			gf_json := GF_Json{}
			if err := json.Unmarshal([]byte(data), &gf_json); err != nil {
				fmt.Printf("解析JSON数据失败 -> %+v\n", err)
				return
			}
			fmt.Println("==================================")
			fmt.Printf("你一共给格林娜花了：%v 元\n", gf_json.User["spend_point"])
			fmt.Printf("格林娜对你的好感为：%v  Lv.%v/30\n", gf_json.Kalina.Favor, gf_json.Kalina.Level)
		}
	}
}

func (gf *GF) loop() {
	for body := range gf.ch {
		if body.Body == nil {
			break
		}
		go gf.build(body)
	}
}

func (gf *GF) onResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp
	}
	resp.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	gf.ch <- response{
		Host: ctx.Req.Host,
		Path: ctx.Req.URL.Path,
		Body: body,
	}

	return resp
}

func (gf *GF) condition() goproxy.ReqConditionFunc {
	return func(req *http.Request, ctx *goproxy.ProxyCtx) bool {
		if strings.HasSuffix(req.Host, "ppgame.com") || strings.HasSuffix(req.Host, "sn-game.txwy.tw")  || strings.HasSuffix(req.Host, "girlfrontline.co.kr") || strings.HasSuffix(req.Host, "sunborngame.com") || strings.HasSuffix(req.Host, "sn-game.txwy.tw") {
			if strings.HasSuffix(req.URL.Path, "/Index/index") || strings.HasSuffix(req.URL.Path, "/Index/getDigitalSkyNbUid") || strings.HasSuffix(req.URL.Path, "/Index/getUidTianxiaQueue") || strings.HasSuffix(req.URL.Path,"/Index/getUidEnMicaQueue"){
				return true
			}
		}
		return false
	}
}

func (gf *GF) getLocalhost() (string, error) {
	conn, err := net.Dial("tcp", "www.baidu.com:80")
	if err != nil {
		return "", errors.WithMessage(err, "连接 www.baidu.com:80 失败")
	}
	host, _, err := net.SplitHostPort(conn.LocalAddr().String())
	if err != nil {
		return "", errors.WithMessage(err, "解析本地主机地址失败")
	}
	return host, nil
}

func path(req *http.Request) string {
	if req.URL.Path == "/" {
		return req.Host
	}
	return req.Host + req.URL.Path
}
