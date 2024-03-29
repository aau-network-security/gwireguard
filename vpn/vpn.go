package wg

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	"github.com/rs/zerolog/log"

	"github.com/aau-network-security/gwireguard/config"
)

// add gRPC connection
// tests
// parse configuration
// add append functionality to conf

const (
	// wireguard should be installed before hand
	wgManageBin = "sudo wg"
	wgQuickBin  = "sudo wg-quick"
)

var (
	// todo: fix configuration variables
	//variablesvariablesΩos.Setenv("FOO", "1")
	configuration, _ = config.NewConfig(os.Getenv("CONFIG_PATH"))
	//configPath       = getConfigPath()
	//configuration, _ = config.NewConfig(configPath)
)

type Interface struct {
	address            string // subnet
	saveConfig         bool
	listenPort         uint32
	privateKey         string
	eth                string
	iName              string
	downInterfacesFile string
}

type Peer struct {
	publicKey  string
	allowedIPs string
	endPoint   string
}

type WGInterface struct {
	Address    string
	ListenPort uint32
	SaveConfig bool
	PrivateKey string
	IPRules    []IPRuleForWG
}

type IPRuleForWG struct {
	WgInterfaceName   string
	HostInterfaceName string
}

// addPeer will add peer to VPN server
// wg set <wireguard-interface-name> <peer-public-key> allowed-ips 192.168.0.2/32
// example <>
func addPeer(nic, publicKey, allowedIPs string) (string, error) {
	cmd := wgManageBin + " set " + nic + " peer " + publicKey + " allowed-ips " + allowedIPs
	//_, err := WireGuardCmd(context.Background(), wgManageBin, "set", nic, publicKey, "allowed-ips", allowedIPs)
	log.Info().Msgf("Adding peer command is %s ", cmd)
	out, err := WireGuardCmd(cmd)
	if err != nil {
		log.Error().Msgf("Error on setting peer on interface %v", err)
		return "Failed", err
	}
	log.Info().Msgf("Add peer output %s", string(out))
	return "Peer " + publicKey + " successfully added", nil
}

// removePeer will remove peer from VPN server
// wg rm <peer-public-key> allowed-ips 192.168.0.2/32
func removePeer(peerPublicKey, ipAddress string) (string, error) {
	log.Debug().Msgf("Peer with publickey { %s } is removing from %s", peerPublicKey, ipAddress)
	cmd := wgManageBin + " rm " + peerPublicKey + " allowed-ips " + ipAddress
	//_, err := WireGuardCmd(context.Background(), wgManageBin, "rm", peerPublicKey, "allowed-ips", ipAddress)
	_, err := WireGuardCmd(cmd)
	if err != nil {
		return "Error", err
	}

	return "Peer " + peerPublicKey + " deleted !", nil
}

// listPeers function basically returns output of executed command,
// this returned data could be improved in order to have structured templating...
func listPeers(interfaceName string) (string, error) {
	// DO NOT return anything if wireguard interface is not given
	if interfaceName == "" {
		return "Error", fmt.Errorf("It is not possible to list peers for empty interface, provide valid interface name !")
	}
	cmd := wgManageBin + " show " + interfaceName
	out, err := WireGuardCmd(cmd)
	if err != nil {
		log.Warn().Msgf("List peers execution error %v", err)
		return "Error", err
	}

	//t := template.Must(template.New("peers").Parse(interfaceName))
	//if err := t.Execute(os.Stdout, string(out)); err != nil {
	//	log.Warn().Msgf("executing template: %v", err)
	//}
	return string(out), err
}

func checkStatus(nicName, publicKey string) (bool, error) {
	var listOfPeers []string
	peerStatus := make(map[string]int)
	cmd := wgManageBin + " show " + nicName + " latest-handshakes"
	out, err := WireGuardCmd(cmd)
	if err != nil {
		return false, err
	}
	outStr := string(out)
	listOfPeers = strings.Split(outStr, "\n")
	for _, v := range listOfPeers {
		peerInfoList := strings.Split(v, "\t")
		if len(peerInfoList) == 2 {
			n, err := strconv.Atoi(peerInfoList[1])
			if err != nil {
				return false, err
			}
			peerStatus[peerInfoList[0]] = n
		}

	}
	if peerStatus[publicKey] == 0 {
		return false, nil
	}
	return true, nil
}

// wg show <name-of-interface>
func nicInfo(nicName string) ([]byte, error) {
	cmd := wgManageBin + " show " + nicName
	log.Info().Msgf("Retrieving configuration of %s ", nicName)
	out, err := WireGuardCmd(cmd)
	if err != nil {
		return []byte("Error: "), err
	}
	return out, nil
}

// all in once
// wg genkey | tee privatekey | wg pubkey > publickey

// wg pubkey < privatekey > publickey
func generatePublicKey(ctx context.Context, privateKeyName, publicKeyName string) error {
	//directory := configuration.WgInterface.Dir
	log.Info().Msgf("Generating public key ...")
	cmd := wgManageBin + " pubkey < " + privateKeyName

	out, err := exec.CommandContext(ctx, "bash", "-c", cmd).Output()
	if err != nil {
		return fmt.Errorf("failed to execute command: %s", cmd)

	}

	if err := writeToFile(publicKeyName, string(out)); err != nil {
		return err
	}
	return nil
}

// wg-quick up wg0
// wg0 configuration file should be exists at /etc/wireguard/
// or the place where docker is mounted
func upDown(nic, cmd string) (string, error) {
	command := wgQuickBin + " " + cmd + " " + nic
	log.Info().Msgf("Interface %s is called to be %s", nic, cmd)
	_, err := WireGuardCmd(command)
	if err != nil {

		return "", fmt.Errorf("failed to execute command: %s error: %v", command, err)
	}

	return "Interface " + nic + " is " + cmd, nil
}

//wg genkey > privatekey
func generatePrivateKey(privateKeyName string) (string, error) {
	cmd := wgManageBin + " genkey"
	log.Info().Msgf("Generating private key with name %s", privateKeyName)
	out, err := WireGuardCmd(cmd)
	if err != nil {
		return "Error on running wg bin, unable to generate private key", fmt.Errorf("GeneratePrivateKey error %v", err)
	}
	log.Info().Msgf("Private key is generated %s", privateKeyName)
	output := strings.Replace(string(out), "\n", "", 1)
	if err := writeToFile(privateKeyName, output); err != nil {
		return "WriteToFile Error ", err
	}
	return output, nil
}

// getContent returns content of privateKey or publicKey depending on keyName
func getContent(keyName string) (string, error) {

	out, err := ioutil.ReadFile(configuration.WgConfig.Dir + keyName)
	if err != nil {
		return "", fmt.Errorf("could not read the file %s err: %v", keyName, err)
	}
	return string(out), nil
}

// will generate configuration file regarding to wireguard interface
func genInterfaceConf(i Interface, confPath string) (string, error) {
	var ipRules []IPRuleForWG
	var hostInterfaces []string

	// and pass it to WGInterface
	if i.downInterfacesFile != "" {
		// read interfaces from a file
		dat, err := ioutil.ReadFile(i.downInterfacesFile)
		if err != nil {
			return "", err
		}
		hostInterfaces = strings.Split(string(dat), ",")
		for _, hostI := range hostInterfaces {
			hostClean := strings.Replace(hostI, "\n", "", -1)
			ipRules = append(ipRules, IPRuleForWG{
				WgInterfaceName:   i.iName,
				HostInterfaceName: hostClean,
			})
		}
	} else {
		ipRules = append(ipRules, IPRuleForWG{
			WgInterfaceName:   i.iName,
			HostInterfaceName: "eth0",
		})
	}

	wgI := WGInterface{
		Address:    i.address,
		ListenPort: i.listenPort,
		SaveConfig: false,
		PrivateKey: i.privateKey,
		IPRules:    ipRules,
	}

	wgConf := createWGIContent(wgI, configuration.WgConfig.WGInterfaceTemplate)

	log.Info().Msgf("Generating interface configuration file for event %s", i.iName)

	if err := writeToFile(confPath+i.iName+".conf", wgConf); err != nil {
		return "GenInterface Error:  ", err
	}
	return i.iName + " configuration saved to " + configuration.WgConfig.Dir, nil
}

func WireGuardCmd(cmd string) ([]byte, error) {
	log.Debug().Msgf("Executing command { %s }", cmd)
	c := exec.Command("/bin/sh", "-c", cmd)
	out, err := c.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return out, nil
}

func createWGIContent(interfaces WGInterface, templatePath string) string {

	var tpl bytes.Buffer
	tmpl := template.Must(template.ParseFiles(templatePath))
	tmpl.Execute(&tpl, interfaces)
	return tpl.String()
}

func writeToFile(filename string, data string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.WriteString(file, data)
	if err != nil {
		return err
	}
	return file.Sync()
}

//helper function to always get the correct config path
func getConfigPath() string {
	dir, err := os.Getwd() // get working directory
	if err != nil {
		log.Error().Msgf("Error getting the working dir %v", err)
	}
	fullPathToconfig := fmt.Sprintf("%s%s", dir, "/config/config.yml")

	return fullPathToconfig
}

func init() {
	getConfigPath()
}
