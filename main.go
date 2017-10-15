package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/garyburd/redigo/redis"
)

const configURL = "https://raw.githubusercontent.com/breadysimon/kube-deploy/master/README.md" // "http://10.214.169.111:30883/report/names.json"

type config struct {
	Bss []string
	Oss []string
	IP  string
}

func getInstruction() string {
	if os.Getenv("windir") != "" {
		return "  1.右键点击文件名\n  2.选择'以管理员身份运行'\n  3.如果有系统安全警告请允许执行"
	} else {
		return "  请运行命令 sudo wdchosts"
	}

}
func showMsg(msg string) {
	fmt.Println("---------------------------------------")
	fmt.Println(msg)
	fmt.Println("\n按回车键返回...")
	fmt.Println("---------------------------------------")
	_, _ = fmt.Scanln()
}
func getConfigDepreciated() *config {
	resp, err := http.Get(configURL)
	if err != nil {
		showMsg("出错!\n\n网络连接失败.\n请确认可以访问万达云公司内网.")
		log.Fatal(err)
	}

	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		showMsg("出错!\n\n读取配置文件失败.")
		log.Fatal(err)
	}
	var conf config
	err = json.Unmarshal(data, &conf)
	if err != nil {
		showMsg("出错\n\n配置文件解析失败.")
		log.Fatal(err)
	}
	return &conf
}
func getConfig(connStr string) *config {
	conn, err := redis.Dial("tcp", connStr)
	if err != nil {
		showMsg("出错!\n\n网络连接失败.\n请确认可以访问万达云公司内网.")
		log.Fatal(err)
	}
	defer conn.Close()
	data, err := redis.Bytes(conn.Do("GET", "names"))
	if err != nil {
		showMsg("出错!\n\n读取配置信息失败.")
		log.Fatal(err)
	}
	log.Printf("getConfig(),redis data:%s", string(data))

	var conf config
	err = json.Unmarshal(data, &conf)
	if err != nil {
		showMsg("出错\n\n配置文件解析失败.")
		log.Fatal(err)
	}
	log.Printf("getConfig(),return:%s", conf)
	return &conf
}
func getHostsFilePath() string {

	windir := os.Getenv("windir")
	if windir == "" {
		return "/etc/hosts"
	} else {
		return filepath.Join(windir, "system32", "drivers", "etc", "hosts")
	}
}
func getTempFilePath() string {
	return filepath.Join(os.TempDir(), "hosts.tmp")
}
func appendHostName(tmpf *os.File, ip string, name string) {
	s := fmt.Sprintf("%s %s.cloud.wanda.cn\n", ip, name)
	_, err := tmpf.WriteString(s)
	if err != nil {
		showMsg("出错!\n\n临时文件写入失败.")
		log.Fatal(err)
	}
}
func makeTempHostsFile(conf *config) string {
	tempFile := getTempFilePath()
	tmpf, err := os.Create(tempFile)
	if err != nil {
		showMsg("出错!\n\n无法创建临时文件.")
		log.Fatal(err)
	}
	defer tmpf.Close()

	f, err := os.Open(getHostsFilePath())
	if err != nil {
		showMsg("出借!\n\n无法读取hosts文件")
		log.Fatal(err)
	}
	defer f.Close()
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			showMsg("出错!\n\n读取hosts文件信息异常.")
			log.Fatal(err)
		}
		s := string(line)
		if strings.Contains(s, ".cloud.wanda.cn") {
			log.Printf("ignore current item: %s", s)
		} else {
			_, err := tmpf.WriteString(s)
			if err != nil {
				showMsg("出错!\n\n临时文件写入失败.")
				log.Fatal(err)
			}
		}
	}

	for _, x := range conf.Bss {
		appendHostName(tmpf, conf.IP, string(x))
	}

	for _, x := range conf.Oss {
		appendHostName(tmpf, conf.IP, string(x))
	}

	return tempFile
}
func replaceHostsFile(tempFile string) {
	hosts := getHostsFilePath()
	err := os.Rename(hosts, hosts+".bak")
	if err != nil {
		showMsg("提示!\n\n需要必要的访问权限,请:\n" + getInstruction())
		log.Fatal(err)
	}
	err = os.Rename(tempFile, hosts)
	if err != nil {
		showMsg("提示!\n需要必要的访问权限,请:\n" + getInstruction())
		log.Fatal(err)
	}
}
func setConfig(connStr string, conf *config) {
	conn, err := redis.Dial("tcp", connStr)
	if err != nil {
		showMsg("出错!\n\n网络连接失败.\n请确认可以访问万达云公司内网.")
		log.Fatal(err)
	}
	defer conn.Close()
	data, err := json.Marshal(conf)
	if err != nil {
		showMsg("出错!\n\n配置信息转换失败.")
		log.Fatal(err)
	}
	_, err = conn.Do("SET", "names", string(data))
	if err != nil {
		showMsg("出错!\n\n配置信息保存失败.")
		log.Fatal(err)
	}
}
func addNames(connStr string, subset string, names []string) {
	conf := getConfig(connStr)
	lst := conf.Bss
	if subset == "oss" {
		lst = conf.Oss
	}
	m := make(map[string]bool)
	for _, x := range lst {
		m[x] = true
	}
	for _, x := range names {
		m[x] = true
	}
	lst = []string{}
	for k, _ := range m {
		lst = append(lst, k)
	}
	if subset == "oss" {
		conf.Oss = lst
	} else {
		conf.Bss = lst
	}
	setConfig(connStr, conf)
}
func setNames(connStr string, subset string, names []string) {
	delNames(connStr, names)

	conf := getConfig(connStr)

	if subset == "oss" {
		conf.Oss = names
	} else {
		conf.Bss = names
	}
	setConfig(connStr, conf)
}

func delNames(connStr string, names []string) {
	conf := getConfig(connStr)
	m := make(map[string]string)
	for _, x := range conf.Bss {
		m[x] = "bss"
	}
	for _, x := range conf.Oss {
		m[x] = "oss"
	}
	for _, x := range names {
		delete(m, x)
	}
	conf.Bss = []string{}
	conf.Oss = []string{}
	for k, v := range m {
		if v == "bss" {
			conf.Bss = append(conf.Bss, k)
		} else {
			conf.Oss = append(conf.Oss, k)
		}
	}
	setConfig(connStr, conf)
}
func showNames(connStr string) {
	conf := getConfig(connStr)
	fmt.Print("BSS:")
	for _, x := range conf.Bss {
		fmt.Printf(" %s", x)
	}

	fmt.Print("\nOSS:")
	for _, x := range conf.Oss {
		fmt.Printf(" %s", x)
	}
	fmt.Println()
}
func main() {
	addOSS := flag.Bool("o", false, "添加OSS子域名列表")
	setOSS := flag.Bool("O", false, "替换OSS子域名列表")
	addBSS := flag.Bool("b", false, "添加BSS子域名列表")
	setBSS := flag.Bool("B", false, "替换BSS子域名列表")
	conn := flag.String("c", "10.214.169.111:31489", "Redis连接参数")
	del := flag.Bool("d", false, "删除子域名列表")
	lst := flag.Bool("l", false, "显示子域名列表")
	flag.Usage = usage

	flag.Parse()

	logfile, _ := os.Create(filepath.Join(os.TempDir(), "hosts.log"))
	defer logfile.Close()
	log.SetOutput(logfile)

	subset := "bss"
	if *addOSS || *setOSS {
		subset = "oss"
	}
	switch {
	case *addBSS || *addOSS:
		addNames(*conn, subset, flag.Args())
		showNames(*conn)
	case *setBSS || *setOSS:
		setNames(*conn, subset, flag.Args())
		showNames(*conn)
	case *del:
		delNames(*conn, flag.Args())
		showNames(*conn)
	case *lst:
		showNames(*conn)
	default:
		if len(flag.Args()) == 0 {
			replaceHostsFile(makeTempHostsFile(getConfig(*conn)))
			showMsg("成功!\n\n已完成hosts文件更新.")
		} else {
			flag.Usage()
		}
	}

}
func usage() {
	fmt.Fprintf(os.Stderr, `-------------------------------------------------
WandaCloud cloud.wanda.cn
子域名配置工具, v1.0
-------------------------------------------------
用法: 
names [-oObBdl] [-c 数据连接] [一个或多个子域名]
无参数时修改本机hosts文件,定义各子域名IP映射.
参数:
`)
	flag.PrintDefaults()
}
