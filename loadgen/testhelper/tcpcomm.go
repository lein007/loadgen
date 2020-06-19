package testhelper

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"time"

	loadgenlib "loadgen.com/loadgen/lib"
)

// TCPComm 表示TCP通讯器的结构。
type TCPComm struct {
	addr string
}

// NewTCPComm 会新建一个TCP通讯器。
func NewTCPComm(addr string) loadgenlib.Caller {
	return &TCPComm{addr: addr}
}

// BuildReq 会构建一个请求。
func (comm *TCPComm) BuildReq() loadgenlib.RawReq {
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
func (comm *TCPComm) Call(req []byte, timeoutNS time.Duration) ([]byte, error) {
	conn, err := net.DialTimeout("tcp", comm.addr, timeoutNS)
	if err != nil {
		return nil, err
	}
	_, err = write(conn, req, DELIM)
	if err != nil {
		return nil, err
	}
	return read(conn, DELIM)
}

// CheckResp 会检查响应内容。
func (comm *TCPComm) CheckResp(
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
	var sresp ServerResp
	err = json.Unmarshal(rawResp.Resp, &sresp)
	if err != nil {
		commResult.Code = loadgenlib.RET_CODE_ERROR_RESPONSE
		commResult.Msg =
			fmt.Sprintf("Incorrectly formatted Resp: %s!\n", string(rawResp.Resp))
		return &commResult
	}
	if sresp.ID != sreq.ID {
		commResult.Code = loadgenlib.RET_CODE_ERROR_RESPONSE
		commResult.Msg =
			fmt.Sprintf("Inconsistent raw id! (%d != %d)\n", rawReq.ID, rawResp.ID)
		return &commResult
	}
	if sresp.Err != nil {
		commResult.Code = loadgenlib.RET_CODE_ERROR_CALEE
		commResult.Msg =
			fmt.Sprintf("Abnormal server: %s!\n", sresp.Err)
		return &commResult
	}
	if sresp.Result != op(sreq.Operands, sreq.Operator) {
		commResult.Code = loadgenlib.RET_CODE_ERROR_RESPONSE
		commResult.Msg =
			fmt.Sprintf(
				"Incorrect result: %s!\n",
				genFormula(sreq.Operands, sreq.Operator, sresp.Result, false))
		return &commResult
	}
	commResult.Code = loadgenlib.RET_CODE_SUCCESS
	commResult.Msg = fmt.Sprintf("Success. (%s)", sresp.Formula)
	return &commResult
}
