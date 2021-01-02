package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
)

type reqmsg struct {
	dstaddr [4]uint8
	dstport [2]uint8
} //定义socks5请求包结构-接收

var connArray [65536]net.Conn //连接池

const bufmax = 1 << 20 //最大缓存区大小

func handlehandshark(w http.ResponseWriter, r *http.Request) {

	var recvbuf [bufmax]byte //接收客户端数据缓冲区
	//var sendbuf [bufmax]byte //发送客户端数据缓存区
	var reqaddr reqmsg //客户端请求地址
	n, _ := r.Body.Read(recvbuf[0:bufmax])

	//get atyp
	atyp := r.Header.Get("x-atyp-2955")
	atypInt, _ := strconv.Atoi(atyp)

	var ip net.IP
	var addrstr string

	switch atypInt {
	case 1:
		ip = recvbuf[0:4]
		addrstr = ip.String()
		reqaddr.dstport[0] = recvbuf[4]
		reqaddr.dstport[1] = recvbuf[5]
	case 3:
		addrstr = fmt.Sprintf("%s", recvbuf[0:n-2])
		reqaddr.dstport[0] = recvbuf[n-2]
		reqaddr.dstport[1] = recvbuf[n-1]
	case 4:
		ip = recvbuf[0:16]
		addrstr = "[" + ip.String() + "]"
		reqaddr.dstport[0] = recvbuf[16]
		reqaddr.dstport[1] = recvbuf[17]
	}

	//获取index值
	index := r.Header.Get("x-index-2955")
	indexInt, _ := strconv.Atoi(index)
	fmt.Println(indexInt)

	//构造目标地址和端口
	//addrstr := fmt.Sprintf("%d.%d.%d.%d", reqaddr.dstaddr[0], reqaddr.dstaddr[1], reqaddr.dstaddr[2], reqaddr.dstaddr[3])
	fmt.Println("this is :", addrstr)
	port := fmt.Sprintf("%d", uint16(reqaddr.dstport[0])<<8|uint16(reqaddr.dstport[1]))
	fmt.Println(port)

	//执行cmd
	conn, _ := net.Dial("tcp", addrstr+":"+port) //执行CONNECT CMD

	if conn != nil {
		connArray[indexInt] = conn
		io.WriteString(w, "good")
		fmt.Println("we get a remote connection")
	} else {
		w.WriteHeader(http.StatusBadRequest)
	}
}

func post(w http.ResponseWriter, r *http.Request) {
	//获取index值
	//var sendbuf [bufmax]byte
	index := r.Header.Get("x-index-2955")
	indexInt, _ := strconv.Atoi(index)
	fmt.Println(indexInt)
	//获取出站连接
	conn := connArray[indexInt]
	//读取client数据并发送至目标网站
	if conn != nil {
		buf, err := ioutil.ReadAll(r.Body)
		fmt.Println("read from client ,the err is :", err)
		n, err := conn.Write(buf)
		fmt.Println("send data to remote ,the err is :", err, ",the data length is :", n)
	}
	/*for {
		n, _ := r.Body.Read(sendbuf[0:bufmax])
		if n <= 0 {
			break
		}
		conn.Write(sendbuf[0:n])
	}*/
	//n, _ := r.Body.Read(sendbuf[0:bufmax])
	//fmt.Println("we got some data from client:", n)
	//conn.Write(sendbuf[0:n])
	r.Body.Close()
	io.WriteString(w, "good")
}

func get(w http.ResponseWriter, r *http.Request) {
	var recvbuf [bufmax]byte
	//获取index值
	index := r.Header.Get("x-index-2955")
	indexInt, _ := strconv.Atoi(index)
	fmt.Println(indexInt)
	//获取入站连接
	conn := connArray[indexInt]
	//读取目标网站数据并发送至client
	if conn != nil {
		n, err := conn.Read(recvbuf[0:bufmax])
		fmt.Println("read data from remote the datalength is :", n, ",the err is :", err)
		if err != nil && n == 0 {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.Write(recvbuf[0:n])
		}
	} else {
		fmt.Println("the remote conn had closed")
		w.WriteHeader(http.StatusBadRequest)
	}

	r.Body.Close()
	return
}

func dns(w http.ResponseWriter, r *http.Request) {
	var recvbuf [bufmax]byte
	buf, _ := ioutil.ReadAll(r.Body)
	conn, _ := net.Dial("udp", "8.8.8.8:53")
	conn.Write(buf)
	n, _ := conn.Read(recvbuf[0:bufmax])
	w.Write(recvbuf[0:n])
}

func main() {
	// Listen on TCP port 8080 on all interfaces.
	port := os.Getenv("PORT")
	var addr string
	if port != "" {
		flag.StringVar(&addr, "addr", ":"+port, "http service address")
	} else {
		flag.StringVar(&addr, "addr", ":8080", "http service address")
	}
	fmt.Println(addr)
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/handshark", handlehandshark)
	http.HandleFunc("/post", post)
	http.HandleFunc("/get", get)
	http.HandleFunc("/dns", dns)
	log.Fatal(http.ListenAndServe(addr, nil))
}
