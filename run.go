package main

import (
	"fmt"
	"runtime"
	"time"

	. "loadgen.com/loadgen"
	loadgenlib "loadgen.com/loadgen/lib"
	testhelper "loadgen.com/loadgen/testhelper"
)

// printDetail 代表是否打印详细结果。
var printDetail = true

func main() {

	fmt.Println("cpu数量：", runtime.NumCPU())
	runtime.GOMAXPROCS(runtime.NumCPU()) //设置了最大的cpu数量
	//请求地址
	serverAddr := "https://vvapi-test.weifanghaoming.cn/v1/auth/test"

	fmt.Println("请求地址：", serverAddr)

	// 初始化载荷发生器。
	pset := ParamSet{
		Caller:     testhelper.NewHTTPComm(serverAddr),
		TimeoutNS:  2000 * time.Millisecond, //超时时长
		LPS:        uint32(2),               //并发数
		DurationNS: 5 * time.Second,         //请求时间
		ResultCh:   make(chan *loadgenlib.CallResult, 100),
	}
	fmt.Printf("初始化 (超时=%v, 并发数=%d, 请求时长=%v)...",
		pset.TimeoutNS, pset.LPS, pset.DurationNS)
	gen, err := NewGenerator(pset)
	if err != nil {
		fmt.Printf("初始化失败: %s\n",
			err)
	}

	// 开始！
	fmt.Println("开始...")
	gen.Start()

	// 显示结果。
	countMap := make(map[loadgenlib.RetCode]int)
	for r := range pset.ResultCh {
		countMap[r.Code] = countMap[r.Code] + 1
		if printDetail {
			fmt.Printf("Result: ID=%d, Code=%d, Msg=%s, Elapse=%v.\n",
				r.ID, r.Code, r.Msg, r.Elapse)
		}
	}

	var total int
	fmt.Printf("状态统计:")
	for k, v := range countMap {
		codePlain := loadgenlib.GetRetCodePlain(k)
		fmt.Printf("  状态码: %s, Count: %d.\n",
			codePlain, v)
		total += v
	}

	fmt.Printf("总数: %d.\n", total)
	successCount := countMap[loadgenlib.RET_CODE_SUCCESS]
	tps := float64(successCount) / float64(pset.DurationNS/1e9)
	fmt.Printf("每秒负载: %d; 每秒处理: %f.\n", pset.LPS, tps)
	time.Sleep(pset.TimeoutNS)
}
