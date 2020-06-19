package testhelper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	loadgenlib "loadgen.com/loadgen/lib"
)

// HTTPComm 表示HTTP通讯器的结构。
type HTTPComm struct {
	addr string
}

// NewHTTPComm 会新建一个HTTP通讯器。
func NewHTTPComm(addr string) loadgenlib.Caller {
	return &HTTPComm{addr: addr}
}

// BuildReq 会构建一个请求。
func (comm *HTTPComm) BuildReq() loadgenlib.RawReq {
	id := time.Now().UnixNano()
	sreq := ServerReq{
		ID: id,
		Operands: []int{
			int(rand.Int31n(1000) + 1),
			int(rand.Int31n(1000) + 1)},
		Operator: func() string {
			return operators[rand.Int31n(100)%4]
		}(),
	}
	bytes, err := json.Marshal(sreq)
	if err != nil {
		panic(err)
	}
	rawReq := loadgenlib.RawReq{ID: id, Req: bytes}
	return rawReq
}

// Call 会发起一次通讯。
func (comm *HTTPComm) Call(req []byte, timeoutNS time.Duration) ([]byte, error) {
	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(netw, addr string) (net.Conn, error) {
				conn, err := net.DialTimeout(netw, addr, timeoutNS)

				if err != nil {
					return nil, err
				}

				return NewTimeoutConn(conn, timeoutNS), nil
			},
			ResponseHeaderTimeout: timeoutNS,
		},
	}

	return comm.read(client, req, DELIM)
}

// CheckResp 会检查响应内容。
func (comm *HTTPComm) CheckResp(
	rawReq loadgenlib.RawReq, rawResp loadgenlib.RawResp) *loadgenlib.CallResult {

	var commResult loadgenlib.CallResult
	commResult.ID = rawResp.ID
	commResult.Req = rawReq
	commResult.Resp = rawResp
	var sreq ServerReq
	err := json.Unmarshal(rawReq.Req, &sreq)
	if err != nil {
		commResult.Code = loadgenlib.RET_CODE_FATAL_CALL
		commResult.Msg =
			fmt.Sprintf("Incorrectly formatted Req: %s!\n", string(rawReq.Req))
		return &commResult
	}
	if res := regexp.MustCompile(`ok`).FindStringSubmatch(string(rawResp.Resp)); len(res) == 0 {
		commResult.Code = loadgenlib.RET_CODE_ERROR_RESPONSE
		commResult.Msg =
			fmt.Sprintf("Incorrectly formatted Resp: %s!\n", string(rawResp.Resp))
		return &commResult
	}

	commResult.Code = loadgenlib.RET_CODE_SUCCESS
	commResult.Msg = fmt.Sprintf("Success. (%d)", rawResp.ID)
	return &commResult
}

// read 会从连接中读数据直到遇到参数delim代表的字节。
func (comm *HTTPComm) read(client *http.Client, content []byte, delim byte) ([]byte, error) {
	var r http.Request
	r.ParseForm()
	r.Form.Add("content", string(content))
	bodystr := strings.TrimSpace(r.Form.Encode())
	request, err := http.NewRequest(http.MethodPost, comm.addr, strings.NewReader(string(bodystr)))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if resp, err := client.Do(request); err != nil {
		return nil, err
	} else {
		defer resp.Body.Close()
		result, err2 := ioutil.ReadAll(resp.Body)
		if err2 != nil {
			return nil, err
		}

		return result, nil
	}
}
