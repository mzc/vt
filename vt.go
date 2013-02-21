package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
)

type Host struct {
	addr   string
	port   string
	domain string
	phost  string
}

var hosts map[string] Host

var extend_user = map[string] string {
	"r" : "root",
	"u" : "ubuntu",
	"m" : "mulisu",
}

func join_str(s ...string) string {
	return strings.Join(s, "")
}

func read_hosts(conf string) (err error) {

	// config file example:
	// [
	//     { "host":"hm3" , "addr":"hm3"          , "domain":""             , "phost":"hm3", "port":"22"},
	//     { "host":"mail", "addr":"hm3-0-mail-32", "domain":"hm3-0-mail-32", "phost":"hm3", "port":"22"},
	//     { "host":"mow" , "addr":"hm3-1-mow-32" , "domain":"hm3-1-mow-32" , "phost":"hm3", "port":"22"}
	// ]
	type H struct {
		Host string   `json:"host"`
		Addr string   `json:"addr"`
		Port string   `json:"port"`
		Domain string `json:"domain"`
		Phost string  `json:"phost"`
	}
	
	b, err := ioutil.ReadFile(conf)
	if err != nil {
		return
	}

	var hs []H
	err = json.Unmarshal(b, &hs)
	if err != nil {
		return
	}

	hosts = make(map[string]Host)
	for _, h := range hs {
		hosts[h.Host] = Host{ h.Addr, h.Port, h.Domain, h.Phost }
	}
	return
}

func is_phost(h Host) bool {
	return h.domain == ""
}

func default_user() string {
	return extend_user["r"]
}

func ls_hosts(addr, port, user string) error {
	uri := join_str("qemu+ssh://", user, "@", addr, ":", port, "/system")
 	cmd := exec.Command("virsh", "-c", uri, "list --all")
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func go_hosts(addr, port, user string) error {
	uri := join_str("qemu+ssh://", user, "@", addr, ":", port, "/system")
	cmd := exec.Command("virsh", "-c", uri)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func view_hosts(addr, port, domain, user string) error {
	uri := join_str("qemu+ssh://", user, "@", addr, ":", port, "/system")
	cmd := exec.Command("virt-viewer", "-c", uri, domain)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func ssh_hosts(addr, port, user, ssh_cmd string) error {
	uri := join_str(user, "@", addr)
	cmd := exec.Command("ssh", "-X", "-p", port, uri, ssh_cmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	return cmd.Run()
}

func show_hosts() {
	fmt.Println("Supported phost:")
	for k, h := range hosts {
		if is_phost(h) {
			fmt.Println(k)
		}
	}
	fmt.Println("\nSupported vhost:")
	for k, h := range hosts {
		if !is_phost(h) {
			fmt.Println(k)
		}
	}
}

func show_users() {
	fmt.Println("Supported user shortcut:")
	for k, u := range extend_user {
		fmt.Println(k, u)
	}
}

func usage(prog string) {
	fmt.Printf("usage: %s <command> <args ...>\n", prog)
	fmt.Println("The commands supported are:")
	fmt.Println("    ls      List domains on a physical")
	fmt.Println("    go      Exec virsh on a physical")
	fmt.Println("    view    Exec virt-viewer for a virtual")
	fmt.Println("    ssh     Ssh to a physical/virtual")
	fmt.Println("    alias   Show host info")
	
	fmt.Println("Examples:")
	fmt.Printf("    %s ls    <phost|vhost>\n"       , prog)
	fmt.Printf("    %s go    <phost|vhost>\n"       , prog)
	fmt.Printf("    %s view  <vhost>\n"             , prog)
	fmt.Printf("    %s ssh   <phost|vhost> <user>\n", prog)
	fmt.Printf("    %s alias <phost|vhost>\n"       , prog)
}

func main() {
	var err error

	args := os.Args
	prog := path.Base(args[0])

	u, err := user.Current()
	if err != nil {
		fmt.Println("Cannot get current user info:", err)
		return
	}

	conf := join_str(u.HomeDir, "/.config/", prog, "/", prog, ".json")
	err = read_hosts(conf)
	if err != nil {
		fmt.Println("Failed to read/parse", conf, ":", err)
		return
	}

	if len(args) < 3 {
		usage(prog)
		return
	}

	op := args[1]
	h, ok := hosts[args[2]]
	if !ok {
		show_hosts()
		return
	}

	switch op {
	case "ls":
		ph := hosts[h.phost]
		err = ls_hosts(ph.addr, ph.port, default_user())
	case "go":
		ph := hosts[h.phost]
		err = go_hosts(ph.addr, ph.port, default_user())
	case "view":
		if is_phost(h) {
			usage(prog)
			return
		}
		ph := hosts[h.phost]
		err = view_hosts(ph.addr, ph.port, h.domain, default_user())
	case "ssh":
		if len(args) < 4 {
			usage(prog)
			return
		}
		user, ok := extend_user[args[3]]
		if !ok {
			show_users()
			return
		}
		var cmd string
		if len(args) == 5 {
			cmd = args[4]
		} else {
			cmd = ""
		}
		err = ssh_hosts(h.addr, h.port, user, cmd)
	case "alias":
		fmt.Println("addr  :", h.addr)
		fmt.Println("port  :", h.port)
		fmt.Println("phost :", h.phost)
		fmt.Println("domain:", h.domain)
	default:
		usage(prog)
	}

	if err != nil {
		fmt.Println(err)
		usage(prog)
	}
}
