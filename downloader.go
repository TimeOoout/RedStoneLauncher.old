package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
	"github.com/winterssy/sreq"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func Download_game(version string, version_name string) bool {

	all_version_file, _ := sreq.Get("https://piston-meta.mojang.com/mc/game/version_manifest.json").Text()
	sum, _ := strconv.Atoi(gjson.Get(all_version_file, "versions.#").String())
	var ver_url string
	for i := 0; i < sum; i++ {
		if gjson.Get(all_version_file, "versions."+strconv.Itoa(i)+".id").String() == version {
			ver_url = gjson.Get(all_version_file, "versions."+strconv.Itoa(i)+".url").String()
			break
		}
	}
	/*创建文件清单*/
	version_file_content, _ := sreq.Get(ver_url).Text()
	os.MkdirAll(".minecraft/versions/"+version_name+"/", os.ModePerm)
	version_file, _ := os.OpenFile(".minecraft/versions/"+version_name+"/"+version+".json", os.O_CREATE|os.O_TRUNC, 0666)
	func() {
		indexassets_waitgroup.Add(1)
		bufWriter := bufio.NewWriter(version_file)
		bufWriter.WriteString(version_file_content)
		bufWriter.Flush()
		defer version_file.Close()
		defer indexassets_waitgroup.Done()
	}()
	/*创建资源索引文件*/
	assetIndex_url := gjson.Get(version_file_content, "assetIndex.url").String()
	assetIndex_file_content, _ := sreq.Get(assetIndex_url).Text()
	_, err := os.Stat(".minecraft/assets/indexes/" + "/" + version + ".json")
	if err == nil {
		f, _ := ioutil.ReadFile(".minecraft/assets/indexes/" + "/" + version + ".json")
		if sha256.Sum256(f) != sha256.Sum256([]byte(assetIndex_file_content)) {
			os.MkdirAll(".minecraft/assets/indexes/", os.ModePerm)
			assetIndex_file, _ := os.OpenFile(".minecraft/assets/indexes/"+"/"+version+".json", os.O_CREATE|os.O_TRUNC, 0666)
			func() {
				indexassets_waitgroup.Add(1)
				bufWriter := bufio.NewWriter(assetIndex_file)
				bufWriter.WriteString(assetIndex_file_content)
				bufWriter.Flush()
				defer assetIndex_file.Close()
				defer indexassets_waitgroup.Done()
			}()
			/*Objects文件下载*/
			obj_num := gjson.Get(assetIndex_file_content, "objects")
			obj_num.ForEach(func(key, value gjson.Result) bool {
				hash := gjson.Get(value.String(), "hash").String()
				go download_obj(hash)
				return true
			})
		}
	} else {
		os.MkdirAll(".minecraft/assets/indexes/", os.ModePerm)
		assetIndex_file, _ := os.OpenFile(".minecraft/assets/indexes/"+"/"+version+".json", os.O_CREATE|os.O_TRUNC, 0666)
		func() {
			indexassets_waitgroup.Add(1)
			bufWriter := bufio.NewWriter(assetIndex_file)
			bufWriter.WriteString(assetIndex_file_content)
			bufWriter.Flush()
			defer assetIndex_file.Close()
			defer indexassets_waitgroup.Done()
		}()
		/*Objects文件下载*/
		obj_num := gjson.Get(assetIndex_file_content, "objects")
		obj_num.ForEach(func(key, value gjson.Result) bool {
			hash := gjson.Get(value.String(), "hash").String()
			go download_obj(hash)
			return true
		})
	}

	/*依赖库文件下载*/
	os.MkdirAll(".minecraft/libraries/", os.ModePerm)
	os.MkdirAll(".minecraft/versions/"+version_name+"/natives", os.ModePerm)
	lib_num, _ := strconv.Atoi(gjson.Get(version_file_content, "libraries.#").String())
	for i := 0; i < lib_num; i++ {
		go download_library(gjson.Get(version_file_content, "libraries."+strconv.Itoa(i)).String(), version_name)
	}
	//下载主文件
	go download_log4j(gjson.Get(version_file_content, "downloads.client.url").String(), version_name, version, gjson.Get(version_file_content, "logging.client.file.url").String())
	go download_main(version_name, version)
	indexassets_waitgroup.Wait()
	return true
}

func download_main(version_name string, v string, retry_times ...int) bool {
	retry := 3
	if len(retry_times) > 0 {
		if retry_times[0] > 0 {
			retry = retry_times[0]
		}
	}
	indexassets_waitgroup.Add(1)
	defer indexassets_waitgroup.Done()
	client := req.New()
	client.SetTimeout(time.Second * 1800)
	asset_content, err := client.Get("https://bmclapi2.bangbang93.com/version/" + v + "/client")
	if err != nil {
		for i := 0; i < retry; i++ {
			asset_content_, err := client.Get("https://bmclapi2.bangbang93.com/version/" + v + "/client")
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
	client := req.New()
	client.SetTimeout(time.Second * 1800)
	log4j_content, err := client.Get(log4j)
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
	client := req.New()
	client.SetTimeout(time.Second * 1800)
	_, err := os.Stat(".minecraft/assets/objects/" + h[0:2] + "/" + h)
	if os.IsNotExist(err) {
		os.MkdirAll(".minecraft/assets/objects/"+h[0:2], os.ModePerm)
		obj_file_content, err := client.Get("http://resources.download.minecraft.net/" + h[0:2] + "/" + h)
		if err != nil {
			for i := 0; i < retry; i++ {
				obj_file_content_, err := client.Get("http://resources.download.minecraft.net/" + h[0:2] + "/" + h)
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
			fmt.Print("\n Write assets files failed! FILE:" + "http://resources.download.minecraft.net" + h[0:2] + "/" + h + " | ERROR:" + erro.Error() + "\n")
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
	client := req.New()
	client.SetTimeout(time.Second * 1800)
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
		if native_url == "" {
			return true
		}
		native_content, err := client.Get(native_url)
		if err != nil {
			for i := 1; i < retry; i++ {
				native_content_, err := client.Get(native_url)
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
		fmt.Print("Write native files failed! FILE:" + native_url + " | ERROR:" + err.Error() + "\n")
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
		lib_content, erro := client.Get(lib_url)
		if erro != nil {
			for i := 0; i < retry; i++ {
				lib_content_, erro := client.Get(lib_url)
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
