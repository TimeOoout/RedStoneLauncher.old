package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func Execute_game(version string, version_name string, username string, java_path string, download_source int, retry_times int, work_path string, version_isolation bool, max_memory ...string) bool {
	indexassets_waitgroup.Wait()
	var current_path, err = os.Getwd()
	if err != nil {
		fmt.Print("Failed to get Current working path!")
		return false
	}
	current_path += "/"
	if work_path != "" {
		current_path = work_path
	}
	indexes, _ := ioutil.ReadFile(current_path + ".minecraft/assets/indexes/" + version + ".json")
	indexes_content := string(indexes)
	indexes_obj := gjson.Get(indexes_content, "objects")
	indexes_obj.ForEach(func(key, value gjson.Result) bool {
		hash := gjson.Get(value.String(), "hash").String()
		download_limit.Add()
		indexassets_waitgroup.Add(1)
		go download_obj(hash, download_source, current_path, retry_times)
		return true
	})
	indexassets_waitgroup.Wait()
	user_name := username
	f, read_err := ioutil.ReadFile(current_path + ".minecraft/versions/" + version_name + "/" + version + ".json")
	if read_err != nil {
		fmt.Print("\n", "Unable to read JSON FILE:", read_err.Error(), "\n")
		return false
	}
	version_json := string(f)
	cp_path_num, _ := strconv.Atoi(gjson.Get(version_json, "libraries.#").String())
	cp_path := ""
	split := ";"
	if system_info != "windows" {
		split = ":"
	}
	for i := 0; i < cp_path_num; i++ {
		if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.classifiers").Exists() == false {
			if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules").Exists() {
				if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.0.action").String() == "allow" {
					if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.1").Exists() {
						if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.1.os.name").String() == system_info && gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.1.action").String() == "allow" {
							cp_path += current_path + ".minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
						} else {
							cp_path += current_path + ".minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
						}
					} else {
						if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.0.os.name").String() == system_info {
							cp_path += current_path + ".minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
						}
					}
				}
			} else {
				cp_path += current_path + ".minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
			}
		}
	}
	cp_path += current_path + ".minecraft\\versions\\" + version_name + "\\" + version + ".jar"
	java_cmd := "&" + java_path
	if gjson.Get(version_json, "arguments").Exists() {

		if system_info == "windows" {
			if gjson.Get(version_json, "arguments.jvm.1.rules.0.action").String() == "allow" {
				java_cmd += "'-XX:HeapDumpPath=MojangTricksIntelDriversForPerformance_javaw.exe_minecraft.exe.heapdump' "
			}
			if gjson.Get(version_json, "arguments.jvm.2.rules.0.action").String() == "allow" {
				java_cmd += "'-Dos.name=Windows 10' '-Dos.version=10.0' "
			}
		}
		if system_info == "darwin" {
			if gjson.Get(version_json, "arguments.jvm.0.rules.0.action").String() == "allow" {
				java_cmd += "'-XstartOnFirstThread' "
			}
		}
		if system_arch == "x86" {
			if gjson.Get(version_json, "arguments.jvm.3.rules.0.action").String() == "allow" {
				java_cmd += "'-Xss1M' "
			}
		}
	} else {
		if system_info == "windows" {
			java_cmd += "'-XX:HeapDumpPath=MojangTricksIntelDriversForPerformance_javaw.exe_minecraft.exe.heapdump' "
		}
		if system_info == "darwin" {
			java_cmd += "'-XstartOnFirstThread' "
		}
		if system_arch == "x86" {
			java_cmd += "'-Xss1M' "
		}
	}

	java_cmd += "'-Djava.library.path=" + "\"" + current_path + ".minecraft\\versions\\" + version_name + "\\natives\"" + "' "
	java_cmd += "'-Dminecraft.launcher.brand=RSL' '-Dminecraft.launcher.version=" + Launcher_Version + "' "
	java_cmd += "'-cp' '" + cp_path + "' "

	if gjson.Get(version_json, "arguments.game.23.rules.0.features.is_demo_user").String() == "true" {
		java_cmd += "'" + gjson.Get(version_json, "arguments.game.23.value").String() + "' "
	}

	java_cmd += "'" + gjson.Get(version_json, "mainClass").String() + "' "
	java_cmd += "'--username' '" + user_name + "' "
	java_cmd += "'--version' '" + version + "' "
	if version_isolation == true {
		java_cmd += "'--gameDir' '" + current_path + ".minecraft\\versions\\" + version_name + "\\" + "' "
	} else {
		java_cmd += "'--gameDir' '" + current_path + ".minecraft\\' "
	}
	java_cmd += "'--assetsDir' '" + current_path + ".minecraft\\assets" + "' "
	java_cmd += "'--assetIndex' '" + version + "' "
	uuid_hash := sha256.New()
	byte_username := []byte(user_name)
	uuid_hash.Write(byte_username)
	uuid_bytes := uuid_hash.Sum(nil)
	uuid_str := hex.EncodeToString(uuid_bytes)
	java_cmd += "'--uuid' '" + uuid_str + "' "
	java_cmd += "'--accessToken' '" + "723ac913833a460ab0cde964c1ae8983" + "' "
	java_cmd += "'--userType' '" + "Legacy" + "' "
	java_cmd += "'--versionType' '" + "RSL" + "' "
	java_cmd += " '--userProperties' '{}' "
	if gjson.Get(version_json, "arguments.game.24.rules.0.features.has_custom_resolution").String() == "true" && gjson.Get(version_json, "arguments.game.24.rules.0.action").String() == "allow" {
		java_cmd += "'--width' " + "'854'" + "' "
		java_cmd += "'--height' " + "'480'" + "' "
	}

	cmd_file, _ := os.OpenFile("command.ps1", os.O_CREATE|os.O_TRUNC, 0666)
	cmd_file.WriteString(java_cmd)
	cmd_file.Close()
	fmt.Print(java_cmd)
	if system_info == "windows" {
		cmd := exec.Command("powershell", java_cmd)
		go cmd.Run()
	} else {
		cmd := exec.Command(java_cmd)
		go cmd.Run()
	}
	for i := 0; i < 1000; i++ {

	}
	return true
}
