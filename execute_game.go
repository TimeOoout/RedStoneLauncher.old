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

func Execute_game(version string, version_name string, username string, java_path string, max_memory ...string) {
	indexes, _ := ioutil.ReadFile(".minecraft/assets/indexes/" + version + ".json")
	indexes_content := string(indexes)
	indexes_obj := gjson.Get(indexes_content, "objects")
	indexes_obj.ForEach(func(key, value gjson.Result) bool {
		hash := gjson.Get(value.String(), "hash").String()
		go download_obj(hash)
		return true
	})
	indexassets_waitgroup.Wait()
	user_name := username
	f, _ := ioutil.ReadFile(".minecraft/versions/" + version_name + "/" + version + ".json")
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
							cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
						} else {
							cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
						}
					} else {
						if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.0.os.name").String() == system_info {
							cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
						}
					}
				}
			} else {
				cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + split
			}
		}
	}
	cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\versions\\" + version_name + "\\" + version + ".jar"
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
		if system_info == "osx" {
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
			//java_cmd += "'-Dos.name=Windows 10' '-Dos.version=10.0'"
		}
		if system_info == "osx" {
			java_cmd += "'-XstartOnFirstThread' "
		}
		if system_arch == "x86" {
			java_cmd += "'-Xss1M' "
		}
	}

	java_cmd += "'-Djava.library.path=" + "\"D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\versions\\" + version_name + "\\natives\"" + "' "
	java_cmd += "'-Dminecraft.launcher.brand=RSL' '-Dminecraft.launcher.version=0.0.1" + "' "
	java_cmd += "'-cp' '" + cp_path + "' "

	if gjson.Get(version_json, "arguments.game.23.rules.0.features.is_demo_user").String() == "true" {
		java_cmd += "'" + gjson.Get(version_json, "arguments.game.23.value").String() + "' "
	}

	java_cmd += "'" + gjson.Get(version_json, "mainClass").String() + "' "
	java_cmd += "'--username' '" + user_name + "' "
	java_cmd += "'--version' '" + version + "' "
	java_cmd += "'--gameDir' '" + "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\versions\\" + version_name + "\\" + "' "
	java_cmd += "'--assetsDir' '" + "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\assets" + "' "
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

	cmd := exec.Command(java_cmd)
	//cmd.Run()
	cmd.Run()
}
