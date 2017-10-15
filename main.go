package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
func getConfig() *config {
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

func main() {
	logfile, _ := os.Create(filepath.Join(os.TempDir(), "hosts.log"))
	defer logfile.Close()
	log.SetOutput(logfile)

	replaceHostsFile(makeTempHostsFile(getConfig()))

	showMsg("成功!\n\n已完成hosts文件更新.")
}
