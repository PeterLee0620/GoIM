// 该程序用于将结构化日志输出转换为易读格式。
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var service string

func init() {
	//定义命令行参数 -service，用于指定过滤的服务名称。
	flag.StringVar(&service, "service", "", "filter which service to see")
	//创建一个通道，用于接收操作系统信号。注册通道以接收中断信号（Ctrl+C）。向父进程发送中断信号，通知父进程程序启动。
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT)
	syscall.Kill(os.Getppid(), syscall.SIGINT)
}

func main() {
	//解析命令行参数。创建一个字符串构建器，用于高效拼接字符串。
	flag.Parse()
	var b strings.Builder
	//将服务名称转换为小写，方便后续比较。创建一个扫描器，从标准输入逐行读取数据。
	service := strings.ToLower(service)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		//获取当前行文本。
		s := scanner.Text()
		//创建一个空的map，用于存储解析后的JSON数据。
		m := make(map[string]any)
		//将当前行的JSON字符串解析到map中。
		err := json.Unmarshal([]byte(s), &m)
		if err != nil {
			if service == "" {
				fmt.Println(s)
			}
			continue
		}

		//如果指定了服务过滤，且当前日志的服务名不匹配，则跳过。
		if service != "" && strings.ToLower(m["service"].(string)) != service {
			continue
		}

		//默认traceID为空UUID。如果日志中包含trace_id字段：将trace_id格式化为字符串。
		traceID := "00000000-0000-0000-0000-000000000000"
		if v, ok := m["trace_id"]; ok {
			traceID = fmt.Sprintf("%v", v)
		}

		// {"time":"2023-06-01T17:21:11.13704718Z","level":"INFO","msg":"startup","service":"SALES-API","GOMAXPROCS":1}

		//重置字符串构建器。按指定顺序拼接日志的主要字段：服务名、时间、文件、日志级别、traceID、消息内容。
		b.Reset()
		b.WriteString(fmt.Sprintf("%s: %s: %s: %s: %s: %s: ",
			m["service"],
			m["time"],
			m["file"],
			m["level"],
			traceID,
			m["msg"],
		))

		// Add the rest of the keys ignoring the ones we already
		// added for the log.
		//遍历日志中所有字段。
		for k, v := range m {
			switch k {
			case "service", "time", "file", "level", "trace_id", "msg":
				continue
			}

			//以 key[value] 格式拼接其他字段。
			b.WriteString(fmt.Sprintf("%s[%v]: ", k, v))
		}

		//获取拼接后的字符串。打印日志，去掉末尾多余的冒号和空格。
		out := b.String()
		fmt.Println(out[:len(out)-2])
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
	}
}
