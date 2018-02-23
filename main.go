package main

import "fmt"
import "net"
import "bufio"
import "strings"
import "strconv"

type LoadStats struct {
    nclients uint64
    bytesin uint64
	bytesout uint64
}

type OVPNClient struct {
	commonName string
	realAddress string //update to ipaddress struct
	bytesReceived uint64
	bytesSent uint64
	connectedSince string //update to date struct
}

type OVPNRoute struct {
	virtualAddress string //update to ipaddress struct
	commonName string
	realAddress string //update to ipaddress struct
	lastRef string //update to date struct
}

type OVPNStatus struct {
	updated string //update to date struct
	clients []OVPNClient
	routingTable []OVPNRoute
	maxQueueLength uint64
}

type OVPNManagementClient struct {
	serverAddress string
}

func closeConn(c net.Conn) {
	//fmt.Fprintf(c, "exit\n") // say bye to server
	c.Close()
}

func (mc OVPNManagementClient) runCommand(cmd string, end byte) (string, error) {
	conn, err := net.Dial("tcp", mc.serverAddress)
	if err != nil {
		println("oops something went wrong connecting ")
		return "", err
	}
	reader := bufio.NewReader(conn)
	reader.ReadString('\n') // read initial login line
	fmt.Fprintf(conn, cmd+"\n") // issue command
	res, err := reader.ReadString(end)
	closeConn(conn)
	if err != nil {
		println("oops something went wrong when reading input ")
		return "",err
	}
	return res,nil
}

func (mc OVPNManagementClient) runCommandUntil(cmd string, end string) (string, error) {
	conn, err := net.Dial("tcp", mc.serverAddress)
	if err != nil {
		print("oops something went wrong connecting ")
		return "", err
	}
	reader := bufio.NewReader(conn)
	reader.ReadString('\n') // read initial login line
	fmt.Fprintf(conn, cmd+"\n") // issue command
	res := ""
	for ok := true; ok; {
		cres, err := reader.ReadString(end[len(end)-1]) //read until you see last char of end
		if err != nil {
			closeConn(conn)
			println("oops something went wrong when reading input ")
			return "",err
		}
		res += cres
		if cres[len(cres)-len(end):] == end { //stop when the last part of our read string is equal to end
			ok = false
		}
	}
	closeConn(conn)
	return res,nil
}

func trimRN(s string) (string) {
	if s[len(s)-2:len(s)] == "\r\n" {
		return s[0:len(s)-2]
	} else {
		return s
	}	
} 

func (mc OVPNManagementClient) GetLoadStats() (LoadStats, error){
	res, err := mc.runCommand("load-stats",'\n')
	if err != nil {
		return LoadStats{},err
	}
	res = trimRN(res)
	res = strings.Split(res, " ")[1]
	var two []string
	two = strings.Split(res, ",")
	for idx, elem := range two {
		two[idx] = strings.Split(elem, "=")[1]
	}
	nc, _ := strconv.ParseUint(two[0], 10, 64)
	bi, _ := strconv.ParseUint(two[1], 10, 64)
	bo, _ := strconv.ParseUint(two[2], 10, 64)
	return LoadStats{nclients: nc, bytesin: bi, bytesout: bo},nil
}

func ovpnClientFromString(s string) (OVPNClient) {
	fields := strings.Split(s,",")
	br, _ := strconv.ParseUint(fields[2],10,64)
	bs, _ := strconv.ParseUint(fields[3],10,64)
	return OVPNClient{commonName: fields[0], realAddress: fields[1], bytesReceived: br, bytesSent: bs, connectedSince: fields[4]}
}

func ovpnRouteFromString(s string) (OVPNRoute) {
	fields := strings.Split(s,",")
	return OVPNRoute{virtualAddress: fields[0], commonName: fields[1], realAddress: fields[2], lastRef: fields[3]}
}

func maxQueueLengthFromString(s string) uint64 {
	val, _ := strconv.ParseUint(strings.Split(s,",")[1],10,64)
	return val
}

func (mc OVPNManagementClient) GetOVPNStatus() (OVPNStatus, error){
	res, err := mc.runCommandUntil("status", "END\r\n")
	if err != nil {
		println(err)
		return OVPNStatus{},err
	}
	lines := strings.Split(res,"\r\n")
	u := strings.Split(lines[1], ",")[1]
	ctu := true
	var nextI int
	var clients []OVPNClient
	for i := 3; ctu; i++  {
		if lines[i] != "ROUTING TABLE" {
			clients = append(clients, ovpnClientFromString(lines[i]))
		} else {
			ctu = false
			nextI = i + 2 //start of routes
		}
	}
	var routes []OVPNRoute
	ctu = true
	for i := nextI; ctu; i++ {
		if lines[i] != "GLOBAL STATS" {
			routes = append(routes, ovpnRouteFromString(lines[i]))
		} else {
			ctu = false
			nextI = i + 1 //start of global stats
		}
	}
	ql := maxQueueLengthFromString(lines[nextI])
	return OVPNStatus{updated: u, clients: clients, routingTable: routes, maxQueueLength: ql},nil
	
}

func main() {
	client:= OVPNManagementClient{serverAddress: "192.168.255.1:42419"}
	
	ls, err := client.GetLoadStats()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%+v\n",ls)
	s, err := client.GetOVPNStatus()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%+v\n",s)
	
}
