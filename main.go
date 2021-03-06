package main

import (
	"bufio"
	"crypto/md5"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/garyburd/redigo/redis"
	"golang.org/x/crypto/ssh/terminal"
)

type config struct {
	Bss    []string
	Oss    []string
	IP     string
	Cipher string
}

func getInstruction() string {
	if os.Getenv("windir") != "" {
		return "  1.右键点击文件名\n  2.选择'以管理员身份运行'\n  3.如果有系统安全警告请允许执行"
	} else {
		return "  请运行命令 sudo ./names"
	}

}
func showMsg(msg string) {
	fmt.Println("---------------------------------------")
	fmt.Println(msg)
	fmt.Println("\n按回车键返回...")
	fmt.Println("---------------------------------------")
	_, _ = fmt.Scanln()
}
func getConfig() *config {
	var err error
	_, err = redisConn.Do("SET", "names", `{"bss":[],"oss":[],"ip":"10.214.169.99","Cipher":"d0970714757783e6cf17b26fb8e2298f"}`, "NX")
	if err != nil {
		showMsg("出错!\n\n初始化配置信息失败.")
		log.Fatal(err)
	}
	data, err := redis.Bytes(redisConn.Do("GET", "names"))
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
	return filepath.Join(os.TempDir(), "hosts")
}
func appendHostName(tmpf *os.File, ip string, name string) {
	s := fmt.Sprintf("%s    %s.cloud.wanda.cn\n", ip, name)
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
	tempf, err := os.Open(getTempFilePath())
	if err != nil {
		showMsg("出错!\n\n临时文件打开失败.")
		log.Fatal(err)
	}
	defer tempf.Close()
	hostf, err := os.OpenFile(getHostsFilePath(), os.O_CREATE|os.O_RDWR, 644)
	if err != nil {
		showMsg("提示!\n需要必要的访问权限,请:\n" + getInstruction())
		log.Fatal(err)
	}
	defer hostf.Close()
	_, copy_err := io.Copy(hostf, tempf)
	if copy_err != nil {
		showMsg("提示!\n需要必要的访问权限,请:\n" + getInstruction())
		log.Fatal(err)
	}
}
func setConfig(conf *config) {
	data, err := json.Marshal(conf)
	if err != nil {
		showMsg("出错!\n\n配置信息转换失败.")
		log.Fatal(err)
	}
	_, err = redisConn.Do("SET", "names", string(data))
	if err != nil {
		showMsg("出错!\n\n配置信息保存失败.")
		log.Fatal(err)
	}
}
func toCollection(conf *config) map[string]string {
	m := make(map[string]string)
	for _, x := range conf.Bss {
		m[x] = "bss"
	}
	for _, x := range conf.Oss {
		m[x] = "oss"
	}
	return m
}
func toLists(m map[string]string, conf *config) {
	conf.Bss = []string{}
	conf.Oss = []string{}

	for k, v := range m {
		if v == "bss" {
			conf.Bss = append(conf.Bss, k)
		} else {
			conf.Oss = append(conf.Oss, k)
		}
	}
	sort.Strings(conf.Bss)
	sort.Strings(conf.Oss)
}
func addNames(subset string, names []string) {
	conf := getConfig()
	m := toCollection(conf)
	for _, x := range names {
		m[x] = subset
	}
	toLists(m, conf)
	setConfig(conf)
}
func setNames(subset string, names []string) {
	delNames(names)

	conf := getConfig()

	if subset == "oss" {
		conf.Oss = names
	} else {
		conf.Bss = names
	}
	setConfig(conf)
}

func delNames(names []string) {
	conf := getConfig()
	m := toCollection(conf)
	for _, x := range names {
		delete(m, x)
	}
	toLists(m, conf)
	setConfig(conf)
}
func showNames() {
	conf := getConfig()
	fmt.Println("Domain: cloud.wanda.cn\nBSS:")
	for _, x := range conf.Bss {
		fmt.Printf("  %s\n", x)
	}

	fmt.Println("\nOSS:")
	for _, x := range conf.Oss {
		fmt.Printf("  %s\n", x)
	}
	fmt.Println()
}

var redisConn redis.Conn

func connect(connStr string) {
	var err error
	redisConn, err = redis.Dial("tcp", connStr)
	if err != nil {
		showMsg("出错!\n\n网络连接失败.\n请确认可以访问万达云公司内网.")
		log.Fatal(err)
	}
}
func disconnect() {
	redisConn.Close()
}
func passwordOK(pass string) bool {
	conf := getConfig()
	ctx := md5.New()
	ctx.Write([]byte(pass))
	cipher := fmt.Sprintf("%x", ctx.Sum(nil))
	if cipher != conf.Cipher {
		return false
	}
	return true
}
func main() {
	addOSS := flag.Bool("o", false, "添加OSS子域名列表")
	//setOSS := flag.Bool("O", false, "替换OSS子域名列表")
	addBSS := flag.Bool("b", false, "添加BSS子域名列表")
	//setBSS := nil //flag.Bool("B", false, "替换BSS子域名列表")
	conn := flag.String("c", "", "服务器`连接参数`")
	del := flag.Bool("d", false, "删除子域名列表")
	lst := flag.Bool("l", false, "显示子域名列表")
	passwd := flag.String("p", "", "修改配置时提供`口令`")
	flag.Usage = usage

	flag.Parse()
	if *conn == "" {
		*conn = "10.214.169.111:31489"
	}
	logfile, _ := os.Create(filepath.Join(os.TempDir(), "hosts.log"))
	defer logfile.Close()
	log.SetOutput(logfile)

	connect(*conn)
	defer disconnect()

	if *addBSS || *addOSS || *del {
		if *passwd == "" {
			fmt.Print("\n口令:")
			bytes, _ := terminal.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			*passwd = string(bytes)
		}
		if !passwordOK(*passwd) {
			showMsg("请使用正确的口令！")
			log.Fatal("error password.")
		}
	}
	subset := "bss"
	if *addOSS { //|| *setOSS
		subset = "oss"
	}
	switch {
	case *addBSS || *addOSS:
		addNames(subset, flag.Args())
		fmt.Println("添加成功!")
		showNames()
		//	case *setBSS || *setOSS:
		//		setNames(subset, flag.Args())
		//		fmt.Println("替换成功!")
		//		showNames()
	case *del:
		delNames(flag.Args())
		fmt.Println("删除成功!")
		showNames()
	case *lst:
		showNames()
	default:
		if len(flag.Args()) == 0 {
			replaceHostsFile(makeTempHostsFile(getConfig()))
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
names [-obdl] [-c 数据连接] [-p 口令] [一个或多个子域名]
无参数时修改本机hosts文件,定义各子域名IP映射.
参数:
`)
	flag.PrintDefaults()
}
