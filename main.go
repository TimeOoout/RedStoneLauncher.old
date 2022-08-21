package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
	"github.com/winterssy/sreq"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// assetsindex_files
var indexassets_waitgroup sync.WaitGroup
var system_info = runtime.GOOS
var system_arch = runtime.GOARCH

func main() {
	//Default返回一个默认的路由引擎
	r := gin.Default()
	req.SetTimeout(time.Second * 40)
	//验证代理是否存在
	r.POST("/is_exist", func(c *gin.Context) {
		fmt.Printf("[SYSTEM]: " + "INFO: Verify presence." + "\n")
		c.JSON(200, gin.H{"INFO": 0})
	})
	//获取所有版本信息
	r.POST("/get_all_versions", func(c *gin.Context) {
		fmt.Printf("[SYSTEM]: " + "INFO: Get all the MineCraft versions." + "\n")
		data := make(map[string]interface{})
		sreq.Get("https://launchermeta.mojang.com/mc/game/version_manifest.json").JSON(&data)
		c.JSON(200, data)
	})
	//获取LiteLoader列表
	r.POST("/get_LiteLoader_list", func(c *gin.Context) {
		data := make(map[string]interface{})
		sreq.Get("https://bmclapi2.bangbang93.com/liteloader/list?mcversion=" + c.PostForm("Version")).JSON(&data)
		c.JSON(200, data)
	})
	//获取forge列表
	r.POST("/get_forge_list", func(c *gin.Context) {
		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/forge/minecraft/" + c.PostForm("Version")).Text()
		c.JSON(200, data)
	})
	//获取forge支持的MineCraft版本
	r.POST("/get_forge_supported_list", func(c *gin.Context) {
		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/forge/minecraft").Text()
		c.JSON(200, data)
	})
	//获取Optifine列表
	r.POST("/get_optifine_list", func(c *gin.Context) {

		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/optifine/" + c.PostForm("Version")).Text()
		c.JSON(200, data)
	})
	//获取Java列表
	r.POST("/get_java_list", func(c *gin.Context) {

		data, _ := sreq.Get("https://bmclapi2.bangbang93.com/java/list").Text()
		c.JSON(200, data)
	})
	//测试堵塞
	r.POST("/download", func(c *gin.Context) {
		mcver := c.PostForm("MCVersion")

		all_version_file, _ := sreq.Get("https://piston-meta.mojang.com/mc/game/version_manifest.json").Text()
		sum, _ := strconv.Atoi(gjson.Get(all_version_file, "versions.#").String())
		var ver_url string
		for i := 0; i < sum; i++ {
			if gjson.Get(all_version_file, "versions."+strconv.Itoa(i)+".id").String() == mcver {
				ver_url = gjson.Get(all_version_file, "versions."+strconv.Itoa(i)+".url").String()
				break
			}
		}
		/*创建文件清单*/
		version_file_content, _ := sreq.Get(ver_url).Text()
		os.MkdirAll(".minecraft/versions/"+c.PostForm("VersionName")+"/", os.ModePerm)
		version_file, _ := os.OpenFile(".minecraft/versions/"+c.PostForm("VersionName")+"/"+mcver+".json", os.O_CREATE|os.O_TRUNC, 0666)
		func() {
			indexassets_waitgroup.Add(1)
			bufWriter := bufio.NewWriter(version_file)
			bufWriter.WriteString(version_file_content)
			bufWriter.Flush()
			defer version_file.Close()
			indexassets_waitgroup.Done()
		}()
		/*创建资源索引文件*/
		assetIndex_url := gjson.Get(version_file_content, "assetIndex.url").String()
		assetIndex_file_content, _ := sreq.Get(assetIndex_url).Text()
		_, err := os.Stat(".minecraft/assets/indexes/" + "/" + mcver + ".json")
		if err == nil {
			f, _ := ioutil.ReadFile(".minecraft/assets/indexes/" + "/" + mcver + ".json")
			if sha256.Sum256(f) != sha256.Sum256([]byte(assetIndex_file_content)) {
				os.MkdirAll(".minecraft/assets/indexes/", os.ModePerm)
				assetIndex_file, _ := os.OpenFile(".minecraft/assets/indexes/"+"/"+mcver+".json", os.O_CREATE|os.O_TRUNC, 0666)
				func() {
					indexassets_waitgroup.Add(1)
					bufWriter := bufio.NewWriter(assetIndex_file)
					bufWriter.WriteString(assetIndex_file_content)
					bufWriter.Flush()
					defer assetIndex_file.Close()
					indexassets_waitgroup.Done()
				}()
				/*Objects文件下载*/
				obj_num := gjson.Get(assetIndex_file_content, "objects")
				obj_num.ForEach(func(key, value gjson.Result) bool {
					hash := gjson.Get(value.String(), "hash").String()
					download_obj(hash)
					return true
				})
			}
		} else {
			os.MkdirAll(".minecraft/assets/indexes/", os.ModePerm)
			assetIndex_file, _ := os.OpenFile(".minecraft/assets/indexes/"+"/"+mcver+".json", os.O_CREATE|os.O_TRUNC, 0666)
			func() {
				indexassets_waitgroup.Add(1)
				bufWriter := bufio.NewWriter(assetIndex_file)
				bufWriter.WriteString(assetIndex_file_content)
				bufWriter.Flush()
				defer assetIndex_file.Close()
				indexassets_waitgroup.Done()
			}()
			/*Objects文件下载*/
			obj_num := gjson.Get(assetIndex_file_content, "objects")
			obj_num.ForEach(func(key, value gjson.Result) bool {
				hash := gjson.Get(value.String(), "hash").String()
				download_obj(hash)
				return true
			})
		}

		/*依赖库文件下载*/
		os.MkdirAll(".minecraft/libraries/", os.ModePerm)
		os.MkdirAll(".minecraft/versions/"+c.PostForm("VersionName")+"/natives", os.ModePerm)
		lib_num, _ := strconv.Atoi(gjson.Get(version_file_content, "libraries.#").String())
		for i := 0; i < lib_num; i++ {
			download_library(gjson.Get(version_file_content, "libraries."+strconv.Itoa(i)).String(), c.PostForm("VersionName"))
		}
		//下载主文件
		download_log4j(gjson.Get(version_file_content, "downloads.client.url").String(), c.PostForm("VersionName"), c.PostForm("MCVersion"), gjson.Get(version_file_content, "logging.client.file.url").String())
		download_main(c.PostForm("VersionName"), c.PostForm("MCVersion"))
		//indexassets_waitgroup.Wait()

	})

	//启动MineCraft
	r.POST("/execute", func(c *gin.Context) {
		java_path := "'D:\\JavaJDK\\bin\\java.exe' "
		version := c.PostForm("MCVersion")
		version_name := c.PostForm("VersionName")
		indexes, _ := ioutil.ReadFile(".minecraft/assets/indexes/" + version + ".json")
		indexes_content := string(indexes)
		indexes_obj := gjson.Get(indexes_content, "objects")
		indexes_obj.ForEach(func(key, value gjson.Result) bool {
			hash := gjson.Get(value.String(), "hash").String()
			download_obj(hash)
			return true
		})
		indexassets_waitgroup.Wait()
		user_name := c.PostForm("UserName")
		f, _ := ioutil.ReadFile(".minecraft/versions/" + version_name + "/" + version + ".json")
		version_json := string(f)
		cp_path_num, _ := strconv.Atoi(gjson.Get(version_json, "libraries.#").String())
		cp_path := ""
		for i := 0; i < cp_path_num; i++ {
			if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.classifiers").Exists() == false {
				if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules").Exists() {

					fmt.Print(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.0.os.name").String())

					if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.0.action").String() == "allow" {
						if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.1").Exists() {
							if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.1.os.name").String() == system_info && gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.1.action").String() == "allow" {
								cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + ";"
							} else {
								cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + ";"
							}
						} else {
							if gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".rules.0.os.name").String() == system_info {
								cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + ";"
							}
						}
					}
				} else {
					cp_path += "D:\\GolangFiles\\RedStoneLauncher\\.minecraft\\libraries\\" + strings.ReplaceAll(gjson.Get(version_json, "libraries."+strconv.Itoa(i)+".downloads.artifact.path").String(), "/", "\\") + ";"
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
		cmd.Run()
	})

	/*
		运行代理
	*/
	r.Run(":30713")
}

func download_main(version_name string, v string, retry_times ...int) bool {
	retry := 3
	if len(retry_times) > 0 {
		if retry_times[0] > 0 {
			retry = retry_times[0]
		}
	}
	asset_content, err := req.Get("https://bmclapi2.bangbang93.com/version/" + v + "/client")
	if err != nil {
		for i := 0; i < retry; i++ {
			asset_content_, err := req.Get("https://bmclapi2.bangbang93.com/version/" + v + "/client")
			if err != nil {
				continue
			}
			asset_content = asset_content_
			goto WRITE
		}
		fmt.Print("\n Download main file failed! FILE:" + "https://bmclapi2.bangbang93.com/version/" + v + "/client" + " | ERROR:" + err.Error() + "\n")
		return false
	}
WRITE:
	erro := asset_content.ToFile(".minecraft/versions/" + version_name + "/" + v + ".jar")
	if erro != nil {
		for i := 0; i < retry; i++ {
			erro := asset_content.ToFile(".minecraft/versions/" + version_name + "/" + v + ".jar")
			if erro != nil {
				continue
			}
			return true
		}
		fmt.Print("\n Write main file failed! FILE:" + "https://bmclapi2.bangbang93.com/version/" + v + "/client" + " | ERROR:" + err.Error() + "\n")
		return false
	}
	return false
}

func download_log4j(url string, version_name string, v string, log4j string, retry_times ...int) bool {
	indexassets_waitgroup.Add(1)
	defer indexassets_waitgroup.Done()
	defer runtime.Gosched()
	retry := 3
	if len(retry_times) > 0 {
		if retry_times[0] > 0 {
			retry = retry_times[0]
		}
	}
	log4j_content, err := req.Get(log4j)
	if err != nil {
		for i := 0; i < retry; i++ {
			if err != nil {
				continue
			}
			break
		}
	}
	if err != nil {
		fmt.Print("\n Download log4j file failed! FILE:" + log4j + " | ERROR:" + err.Error() + "\n")
		return false
	}
	erro := log4j_content.ToFile(".minecraft/versions/" + version_name + "/log4j2.xml")
	if erro != nil {
		for i := 0; i < retry; i++ {
			erro := log4j_content.ToFile(".minecraft/versions/" + version_name + "/log4j2.xml")
			if erro != nil {
				continue
			}
			return true
		}
		fmt.Print("\n Write log4j file failed! FILE:" + log4j + " | ERROR:" + erro.Error() + "\n")
		return false
	}

	return true
}

func download_obj(h string, retry_times ...int) bool {
	indexassets_waitgroup.Add(1)
	defer indexassets_waitgroup.Done()
	defer runtime.Gosched()
	retry := 3
	if len(retry_times) > 0 {
		if retry_times[0] > 0 {
			retry = retry_times[0]
		}
	}
	_, err := os.Stat(".minecraft/assets/objects/" + h[0:2] + "/" + h)
	if os.IsNotExist(err) {
		os.MkdirAll(".minecraft/assets/objects/"+h[0:2], os.ModePerm)
		obj_file_content, err := req.Get("http://resources.download.minecraft.net/" + h[0:2] + "/" + h)
		if err != nil {
			for i := 0; i < retry; i++ {
				obj_file_content_, err := req.Get("http://resources.download.minecraft.net/" + h[0:2] + "/" + h)
				if err != nil {
					continue
				}
				obj_file_content = obj_file_content_
				goto WRITE
			}
			fmt.Print("\n Download assets files failed! FILE:" + "http://resources.download.minecraft.net/" + h[0:2] + "/" + h + " | ERROR:" + err.Error() + "\n")
			return false
		}
	WRITE:
		erro := obj_file_content.ToFile(".minecraft/assets/objects/" + h[0:2] + "/" + h)
		if erro != nil {
			for i := 0; i < retry; i++ {
				erro := obj_file_content.ToFile(".minecraft/assets/objects/" + h[0:2] + "/" + h)
				if erro != nil {
					continue
				}
				return true
			}
			fmt.Print("\n Write assets files failed! FILE:" + "http://resources.download.minecraft.net/" + h[0:2] + "/" + h + " | ERROR:" + erro.Error() + "\n")
			return false
		}

	}
	return true
}

func download_library(h string, v string, retry_times ...int) bool {
	retry := 3
	if len(retry_times) > 0 {
		if retry_times[0] > 0 {
			retry = retry_times[0]
		}
	}

	indexassets_waitgroup.Add(1)
	defer indexassets_waitgroup.Done()
	defer runtime.Gosched()
	path_ := gjson.Get(h, "downloads.artifact.path").String()
	lib_url := gjson.Get(h, "downloads.artifact.url").String()
	path_list := strings.Split(path_, "/")
	lib_filename := path_list[len(path_list)-1]
	path_list = path_list[:len(path_list)-1]
	path := strings.Join(path_list, "/")

	if gjson.Get(h, "downloads.classifiers").String() != "" {
		var native_url string
		if system_info == "windows" {
			native_url = gjson.Get(h, "downloads.classifiers.natives-windows.url").String()
		} else if system_info == "darwin" {
			native_url = gjson.Get(h, "downloads.classifiers.natives-osx.url").String()
		} else {
			native_url = gjson.Get(h, "downloads.classifiers.natives-linux.url").String()
		}
		native_content, err := req.Get(native_url)
		if err != nil {
			for i := 1; i < retry; i++ {
				native_content_, err := req.Get(native_url)
				if err != nil {
					continue
				}
				native_content = native_content_
				goto WRITE
			}
		}
	WRITE:
		if err != nil {
			fmt.Print("\n Download native files failed! FILE:" + native_url + " | ERROR:" + err.Error() + "\n")
			return false
		}
		uuid_hash := sha256.New()
		byte_username := []byte(native_url)
		uuid_hash.Write(byte_username)
		uuid_bytes := uuid_hash.Sum(nil)
		uuid_str := hex.EncodeToString(uuid_bytes)
		for i := 0; i < retry; i++ {
			if native_content.ToFile(".minecraft/versions/"+v+"/natives/"+uuid_str) != nil {
				continue
			}
			goto UNZIP
		}
		fmt.Print("Write native files failed! FILE:" + lib_url + " | ERROR:" + err.Error() + "\n")
		return false
	UNZIP:
		Unzip(".minecraft/versions/"+v+"/natives/"+uuid_str, ".minecraft/versions/"+v+"/natives/")
		os.Remove(".minecraft/versions/" + v + "/natives/" + uuid_str)
		files, _ := ioutil.ReadDir(".minecraft/versions/" + v + "/natives/")
		for _, f := range files {
			if f.Name()[len(f.Name())-4:] == ".git" {
				os.Remove(".minecraft/versions/" + v + "/natives/" + f.Name())
			} else if f.Name()[len(f.Name())-5:] == ".sha1" {
				os.Remove(".minecraft/versions/" + v + "/natives/" + f.Name())
			}
		}

	} else {
		os.MkdirAll(".minecraft/libraries/"+path, os.ModePerm)
		lib_content, erro := req.Get(strings.ReplaceAll(lib_url, "https://libraries.minecraft.net/", "https://bmclapi2.bangbang93.com/maven/"))
		if erro != nil {
			for i := 0; i < retry; i++ {
				lib_content_, erro := req.Get(strings.ReplaceAll(lib_url, "https://libraries.minecraft.net/", "https://bmclapi2.bangbang93.com/maven/"))
				if erro != nil {
					continue
				}
				lib_content = lib_content_
				goto WRITE_LIB
			}
			fmt.Print("Download lib failed! FILE:" + lib_url + "| ERROR:" + erro.Error() + "\n")
			return false
		}
		//lib_file, _ := os.OpenFile(".minecraft/libraries/"+path+"/"+lib_filename, os.O_CREATE|os.O_TRUNC, 0666)
		//lib_file.Close()
	WRITE_LIB:
		errorr := lib_content.ToFile(".minecraft/libraries/" + path + "/" + lib_filename)
		if errorr != nil {
			for i := 0; i < retry; i++ {
				errorr := lib_content.ToFile(".minecraft/libraries/" + path + "/" + lib_filename)
				if errorr != nil {
					continue
				}
				return true
			}
			fmt.Print("Write lib failed! FILE:" + lib_url + " | ERROR:" + errorr.Error() + "\n")
			return false
		}
	}
	return true
}

func Unzip(zipFile string, destDir string) error {
	zipReader, err := zip.OpenReader(zipFile)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
		} else {
			if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
				return err
			}

			inFile, err := f.Open()
			if err != nil {
				return err
			}
			defer inFile.Close()

			outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer outFile.Close()

			_, err = io.Copy(outFile, inFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
