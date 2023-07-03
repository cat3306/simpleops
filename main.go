package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"
)

var (
	list []opsModel
	bash = "/bin/bash"
)

type opsModel struct {
	NameSpace string       `json:"name_space"`
	Servers   []serverConf `json:"servers"`
}
type serverConf struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	ServerPath string `json:"server_path"`
}

func initConf() {
	u, err := os.UserHomeDir()
	if err != nil {
		panic(err)
		return
	}
	u = path.Join(u, ".ops.json")
	b, err := ioutil.ReadFile(u)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &list)
	if err != nil {
		panic(err)
	}
	cnt := 1
	for i := 0; i < len(list); i++ {
		for j := 0; j < len(list[i].Servers); j++ {
			list[i].Servers[j].Id = cnt
			cnt++
		}
	}
}
func main() {
	initConf()
	app := cli.App{
		Name:        "ops",
		Usage:       "a ops application",
		Description: "manage ops",
		Commands:    commands(),
	}
	if err := app.Run(os.Args); err != nil {
		fmt.Println(err)
	}
}
func commands() cli.Commands {
	return cli.Commands{
		&cli.Command{
			Name:        "get",
			Usage:       "get server",
			Subcommands: getCommands(),
		},
		&cli.Command{
			Name:        "deploy",
			Usage:       "deploy server",
			Subcommands: deployCommands(),
		},
		&cli.Command{
			Name:   "create",
			Usage:  "simpleops create yebao-dev server-name server-path",
			Action: createConf,
		},
		&cli.Command{
			Name:        "delete",
			Usage:       "simpleops delete yebao-dev server-name",
			Subcommands: deleteConf(),
		},
	}
}
func deleteConf() cli.Commands {
	cs := cli.Commands{
	}
	for _, v := range list {
		cs = append(cs, &cli.Command{
			Name:   v.NameSpace,
			Usage:  "delete server input server-name",
			Action: serversDelete,
		})
	}
	return cs
}
func serversDelete(ctx *cli.Context) error {
	server := ctx.Args().First()
	if server==""{
		return errors.New("args err")
	}
	for i := 0; i < len(list); i++ {
		if list[i].NameSpace == ctx.Command.Name {
			newList := make([]serverConf, 0)
			find := false
			for j := 0; j < len(list[i].Servers); j++ {
				if list[i].Servers[j].Name == server {
					find = true
					continue
				}
				newList = append(newList, list[i].Servers[j])
			}
			if !find {
				return errors.New("server name not found")
			}
			list[i].Servers = newList
			break
		}
	}
	return modifyConf()
}

// simpleops create yebao-dev server-name server-path
func createConf(ctx *cli.Context) error {
	if ctx.Args().Len() != 3 {
		return errors.New("create Args err")
	}
	ns := ctx.Args().First()
	p := ctx.Args().Get(2)
	name := ctx.Args().Get(1)
	var find bool
	for i := 0; i < len(list); i++ {
		if ns == list[i].NameSpace {
			for j := 0; j < len(list[i].Servers); j++ {
				if list[i].Servers[j].ServerPath == p || list[i].Servers[j].Name == name {
					return errors.New("repetitive server name or server path")
				}
			}
			list[i].Servers = append(list[i].Servers, serverConf{
				Name:       name,
				ServerPath: p,
			})
			find = true
			break
		}
	}
	if !find {
		list = append(list, opsModel{
			NameSpace: ns,
			Servers: []serverConf{
				{
					Name:       name,
					ServerPath: p,
				},
			},
		})
	}
	return modifyConf()
}
func modifyConf() error {
	raw, err := json.Marshal(list)
	if err != nil {
		return err
	}
	u, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	u = path.Join(u, ".ops.json")
	err = ioutil.WriteFile(u, raw, 0644)
	if err != nil {
		return err
	}
	return err
}
func deployCommands() cli.Commands {
	cs := cli.Commands{
	}
	for _, v := range list {
		cs = append(cs, &cli.Command{
			Name:   v.NameSpace,
			Usage:  "deploy server input server or id",
			Action: serversDeploy,
		})
	}
	return cs
}
func serversDeploy(ctx *cli.Context) error {
	server := ctx.Args().First()
	tmp := make([]serverConf, 0)
	for _, v := range list {
		if v.NameSpace == ctx.Command.Name {
			tmp = append(tmp, v.Servers...)
			break
		}
	}
	var serv serverConf
	for _, v := range tmp {
		if v.Name == server {
			serv = v
			break
		}
	}
	if serv.Id == 0 {
		return errors.New("not found")
	}
	fName := serv.ServerPath + "/deploy.yaml"
	err, newImage := modifyYaml(fName)
	if err != nil {
		return err
	}
	err = k8sDev(serv.ServerPath, newImage, fName)
	if err != nil {
		return err
	}
	return nil
}

func k8sDev(path string, image string, fName string) error {
	s := fmt.Sprintf("cd %s && docker build -t %s .", path, image)
	fmt.Println(s)
	str, err := UnixCmd(s)
	fmt.Println(str)
	if err != nil {
		return err
	}

	//
	s = fmt.Sprintf("docker push %s", image)
	fmt.Println(s)
	str, err = UnixCmd(s)
	fmt.Println(str)
	if err != nil {
		return err
	}
	//
	s = fmt.Sprintf("kubectl apply -f  %s", fName)
	str, err = UnixCmd(s)
	fmt.Println(str)
	return err
}
func UnixCmd(arg string, timeoutArgs ...time.Duration) (string, error) {
	timeout := 3 * time.Second
	if len(arg) == 0 {
		return "", errors.New("arg empty")
	}
	if len(timeoutArgs) > 0 {
		timeout = timeoutArgs[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	args := []string{"-c"}
	args = append(args, arg)
	cmd := exec.CommandContext(ctx, bash, args...)
	raw, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(raw)), err
}

//func StringToBytes(s string) []byte {
//	return *(*[]byte)(unsafe.Pointer(
//		&struct {
//			string
//			Cap int
//		}{s, len(s)},
//	))
//}
//
//// BytesToString converts byte slice to string without a memory allocation.
//func BytesToString(b []byte) string {
//	return *(*string)(unsafe.Pointer(&b))
//}

func modifyYaml(fName string) (error, string) {

	raw, err := ioutil.ReadFile(fName)
	if err != nil {
		return err, ""
	}

	yamlMap := make(map[string]interface{})
	err = yaml.Unmarshal(raw, &yamlMap)
	if err != nil {
		return err, ""
	}
	var target map[string]interface{}
	if v, ok := yamlMap["spec"].(map[string]interface{}); ok {
		if vv, ok := v["template"].(map[string]interface{}); ok {
			if vvv, ok := vv["spec"].(map[string]interface{}); ok {
				if vvvv, ok := vvv["containers"]; ok {
					if vvvvv, ok := vvvv.([]interface{}); ok {
						for _, vvvvvv := range vvvvv {
							if vvvvvvv, ok := vvvvvv.(map[string]interface{}); ok {
								target = vvvvvvv
								break
							}
						}
					}
				}
			}
		}
	}
	if target == nil {
		return errors.New("yaml err"), ""
	}
	image := target["image"].(string)
	imageStr := strings.Split(image, ":")
	if len(imageStr) < 2 {
		return fmt.Errorf("yaml err image:%s", image), ""
	}
	imageStr[1] = fmt.Sprintf("v%v", time.Now().UnixMilli())
	newImage := strings.Join(imageStr, ":")
	target["image"] = newImage
	raw, err = yaml.Marshal(yamlMap)
	if err != nil {
		return err, ""
	}
	return ioutil.WriteFile(fName, raw, 0644), newImage
}

func getCommands() cli.Commands {
	cs := cli.Commands{
	}
	for _, v := range list {
		cs = append(cs, &cli.Command{
			Name:   v.NameSpace,
			Usage:  "show namespace server",
			Action: serversByNs,
		})
	}
	return cs
}
func serversByNs(ctx *cli.Context) error {
	first := ctx.Args().First()
	tmp := make([]serverConf, 0)
	for _, v := range list {
		if v.NameSpace == ctx.Command.Name {
			tmp = append(tmp, v.Servers...)
			break
		}

	}
	if first != "" {
		tmp1 := make([]serverConf, 0)
		for _, v := range tmp {
			if strings.Contains(v.Name, first) {
				tmp1 = append(tmp1, v)
			}
		}
		tmp = tmp1
	}
	fmt.Println(Table(tmp))
	return nil
}
