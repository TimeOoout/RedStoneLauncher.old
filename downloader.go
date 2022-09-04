package main

import (
	"archive/zip"
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type GoLimit struct {
	ch chan int
}

func NewGoLimit(max int) *GoLimit {
	return &GoLimit{ch: make(chan int, max)}
}

func (g *GoLimit) Add() {
	g.ch <- 1
}

func (g *GoLimit) Done() {
	<-g.ch
}

func Download_game(version string, version_name string, retry_times int, download_source int, work_path string, threads int) string {
	//获取版本列表
	source := MineCraftSource
	if download_source == 1 {
		source = BMCL
	} else if download_source == 2 {
		source = MCBBS
	}
	ver_list := gjson.Get(source, "VersionList_v1").String()
	all_version_file, mis := req.Get(ver_list)
	if mis != nil {
		for i := 0; i < retry_times; i++ {
			all_version_file, mis = req.Get(ver_list)
			if mis == nil {
				break
			}
		}
	}
	if mis != nil {
		Info("Failed to get version list. Error:" + mis.Error())
		return "Failed to get version list. Error:" + mis.Error()
	}
	all_version_file_text := all_version_file.String()
	sum, _ := strconv.Atoi(gjson.Get(all_version_file_text, "versions.#").String())
	var ver_url string
	for i := 0; i < sum; i++ {
		if gjson.Get(all_version_file_text, "versions."+strconv.Itoa(i)+".id").String() == version {
			ver_url = gjson.Get(all_version_file_text, "versions."+strconv.Itoa(i)+".url").String()
			goto GET_VERSION_JSON
		}
	}
	Info("Unable to find this version.")
	return "Unable to find this version."
GET_VERSION_JSON:
	/*创建文件清单*/
	version_file_content, miss := req.Get(ver_url)
	if miss != nil {
		for i := 0; i < retry_times; i++ {
			version_file_content, miss = req.Get(ver_url)
			if miss == nil {
				break
			}
		}
		if miss != nil {
			Info("Failed to download version_json_file. Error:" + miss.Error())
			return "Failed to download version_json_file. Error:" + miss.Error()
		}
	}
	os.MkdirAll(work_path+".minecraft/versions/"+version_name+"/", os.ModePerm)

	ver_json := version_file_content.String()

	main_url := gjson.Get(ver_json, "downloads.client.url").String()
	if download_source == 1 {
		main_url = "https://bmclapi2.bangbang93.com/version/" + version + "/client"
	} else if download_source == 2 {
		main_url = "https://download.mcbbs.net/version/" + version + "/client"
	}
	//原来的下载client
	go download_file(main_url, work_path+".minecraft/versions/"+version_name+"/"+version+".jar", threads, retry_times)

	er := version_file_content.ToFile(work_path + ".minecraft/versions/" + version_name + "/" + version + ".json")
	if er != nil {
		Info("Failed to write version_json_file. Error:" + er.Error())
		return "Failed to write version_json_file. Error:" + er.Error()
	}

	strings.ReplaceAll(ver_json, gjson.Get(MineCraftSource, "Index").String(), gjson.Get(source, "Index").String())
	strings.ReplaceAll(ver_json, gjson.Get(MineCraftSource, "Launcher").String(), gjson.Get(source, "Launcher").String())
	strings.ReplaceAll(ver_json, gjson.Get(MineCraftSource, "Libraries").String(), gjson.Get(source, "Libraries").String())
	strings.ReplaceAll(ver_json, gjson.Get(MineCraftSource, "Assets").String(), gjson.Get(source, "Assets").String())
	/*创建资源索引文件*/
	assetIndex_url := gjson.Get(ver_json, "assetIndex.url").String()
	assetIndex_file_content, misss := req.Get(assetIndex_url)

	if misss != nil {
		for i := 0; i < retry_times; i++ {
			assetIndex_file_content, misss = req.Get(assetIndex_url)
			if misss == nil {
				break
			}
		}
		if misss != nil {
			Info("Failed to download Index_file. Error:" + misss.Error())
			return "Failed to download Index_file. Error:" + misss.Error()
		}
	}
	index_content := assetIndex_file_content.String()
	index_content = strings.ReplaceAll(index_content, gjson.Get(MineCraftSource, "Assets").String(), gjson.Get(source, "Assets").String())
	_, err := os.Stat(work_path + ".minecraft/assets/indexes/" + "/" + version + ".json")
	if err == nil {
		f, _ := ioutil.ReadFile(work_path + ".minecraft/assets/indexes/" + "/" + version + ".json")
		if sha256.Sum256(f) != sha256.Sum256([]byte(index_content)) {
			os.MkdirAll(work_path+".minecraft/assets/indexes/", os.ModePerm)
			assetIndex_file, _ := os.OpenFile(work_path+".minecraft/assets/indexes/"+"/"+version+".json", os.O_CREATE|os.O_TRUNC, 0666)
			func() {
				bufWriter := bufio.NewWriter(assetIndex_file)
				bufWriter.WriteString(index_content)
				bufWriter.Flush()
				defer assetIndex_file.Close()
			}()
			/*Objects文件下载*/
			obj_num := gjson.Get(index_content, "objects")
			os.MkdirAll(work_path+".minecraft/assets/objects/", os.ModePerm)
			obj_num.ForEach(func(key, value gjson.Result) bool {
				hash := gjson.Get(value.String(), "hash").String()
				download_limit.Add()
				indexassets_waitgroup.Add(1)
				go download_obj(hash, download_source, work_path, threads, retry_times)
				return true
			})
		}
	} else {
		os.MkdirAll(work_path+".minecraft/assets/indexes/", os.ModePerm)
		assetIndex_file, _ := os.OpenFile(work_path+".minecraft/assets/indexes/"+"/"+version+".json", os.O_CREATE|os.O_TRUNC, 0666)
		func() {
			bufWriter := bufio.NewWriter(assetIndex_file)
			bufWriter.WriteString(index_content)
			bufWriter.Flush()
			defer assetIndex_file.Close()
		}()
		/*Objects文件下载*/
		obj_num := gjson.Get(index_content, "objects")
		os.MkdirAll(work_path+".minecraft/assets/objects/", os.ModePerm)
		obj_num.ForEach(func(key, value gjson.Result) bool {
			hash := gjson.Get(value.String(), "hash").String()
			download_limit.Add()
			indexassets_waitgroup.Add(1)
			go download_obj(hash, download_source, work_path, threads, retry_times)
			return true
		})
	}
	/*依赖库文件下载*/
	os.MkdirAll(work_path+".minecraft/libraries/", os.ModePerm)
	os.MkdirAll(work_path+".minecraft/versions/"+version_name+"/natives", os.ModePerm)
	lib_num, _ := strconv.Atoi(gjson.Get(ver_json, "libraries.#").String())
	for i := 0; i < lib_num; i++ {
		download_limit.Add()
		indexassets_waitgroup.Add(1)
		go download_library(gjson.Get(ver_json, "libraries."+strconv.Itoa(i)).String(), version_name, work_path, threads, retry_times)
	}
	go download_file(gjson.Get(ver_json, "logging.client.file.url").String(), work_path+version_name+"/log4j2.xml", threads, retry_times)
	indexassets_waitgroup.Wait()
	return ""
}

func download_obj(h string, download_source int, work_path string, threads int, retry_times ...int) bool {
	defer indexassets_waitgroup.Done()
	defer runtime.Gosched()
	defer download_limit.Done()

	retry := 3
	if len(retry_times) > 0 {
		if retry_times[0] > 0 {
			retry = retry_times[0]
		}
	}
	source := ""
	if download_source == 1 {
		source = BMCL
	} else if download_source == 2 {
		source = MCBBS
	} else {
		source = MineCraftSource
	}
	obj_url := gjson.Get(source, "Assets").String()
	client := req.New()
	client.SetTimeout(time.Second * 60)
	_, errr := os.Stat(work_path + ".minecraft/assets/objects/" + h[0:2] + "/" + h)
	if os.IsNotExist(errr) {
		os.MkdirAll(work_path+".minecraft/assets/objects/"+h[0:2], os.ModePerm)
		errrr := download_file(obj_url+h[0:2]+"/"+h, work_path+".minecraft/assets/objects/"+h[0:2]+"/"+h, threads, retry)
		if errrr != nil {
			Info(errrr.Error())
			return false
		}
		return true
	}
	return true
}

func download_library(h string, v string, work_path string, threads int, retry_times ...int) bool {
	retry := 3
	if len(retry_times) > 0 {
		if retry_times[0] > 0 {
			retry = retry_times[0]
		}
	}

	defer download_limit.Done()

	client := req.New()
	client.SetTimeout(time.Second * 180)
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
		uuid_hash := sha256.New()
		byte_username := []byte(native_url)
		uuid_hash.Write(byte_username)
		uuid_bytes := uuid_hash.Sum(nil)
		uuid_str := hex.EncodeToString(uuid_bytes)
		errr := download_file(native_url, work_path+".minecraft/versions/"+v+"/natives/"+uuid_str, threads, retry)
		if errr != nil {
			Info("\nDownload native file failed: " + native_url + "\n")
			return false
		}
		Unzip(work_path+".minecraft/versions/"+v+"/natives/"+uuid_str, work_path+".minecraft/versions/"+v+"/natives/")
		os.Remove(work_path + ".minecraft/versions/" + v + "/natives/" + uuid_str)
		files, _ := ioutil.ReadDir(work_path + ".minecraft/versions/" + v + "/natives/")
		for _, f := range files {
			if f.Name()[len(f.Name())-4:] == ".git" {
				os.Remove(work_path + ".minecraft/versions/" + v + "/natives/" + f.Name())
			} else if f.Name()[len(f.Name())-5:] == ".sha1" {
				os.Remove(work_path + ".minecraft/versions/" + v + "/natives/" + f.Name())
			}
		}

	} else {
		os.MkdirAll(work_path+".minecraft/libraries/"+path, os.ModePerm)
		errr := download_file(lib_url, work_path+".minecraft/libraries/"+path+"/"+lib_filename, threads, retry)
		if errr != nil {
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

/*
新写的协程下载
*/

func download_file(url string, filename string, threads int, retry_times int) error {
	//defer indexassets_waitgroup.Done()
	if threads < 1 {
		return errors.New("The wrong number of threads was given.")
	}
	if retry_times < 1 {
		return errors.New("The wrong number of retry_times was given.")
	}
	e := threadedDownload(url, threads, filename)
	if e != nil {
		if retry_times > 0 {
			return download_file(url, filename, threads-1, retry_times)
		} else {
			return errors.New("Unable to download Client : " + url + " | Error:" + e.Error())
		}
	}
	return nil
}

func threadedDownload(url string, threads int, filename string) error {

	client := http.Client{}

	resp, err := client.Head(url)
	if err != nil {
		return err
	}
	contentLength := resp.Header.Get("Content-Length")
	ranges := resp.Header.Get("Accept-Ranges")
	if contentLength == "" {
		Info("Content length not specified")
		return errors.New("Content length not specified")
	} else {
		if ranges != "bytes" {
			//Info("Server does not accept byte ranges")
			//return errors.New("Server does not accept byte ranges")
			client_ := req.New()
			resp_, e := client_.Get(url)
			if e != nil {
				return errors.New("Failed to get file :" + url + " | Error:" + e.Error())
			}
			e_ := resp_.ToFile(filename)
			if e_ != nil {
				return errors.New("Failed to download file :" + url + " | Error:" + e_.Error())
			}
			return nil
		} else {
			length, _ := strconv.Atoi(contentLength)
			size := length / threads
			remainder := length % threads
			wg := &sync.WaitGroup{}
			os.Remove(filename)
			for i := 0; i < threads; i++ {
				wg.Add(1)

				start := i * size
				end := (i + 1) * size

				if i == threads-1 {
					end += remainder
				}
				go func(start, end, i int) error {
					thr_req, err := http.NewRequest("GET", url, nil)
					if err != nil {
						wg.Done()
						return err
					}
					byteRange := fmt.Sprintf("bytes=%d-%d", start, end-1)
					thr_req.Header.Add("Range", byteRange)
					resp, err := client.Do(thr_req)
					if err != nil {
						wg.Done()
						return err
					}
					defer resp.Body.Close()
					//Info("Thread: %d Reading response body", i)
					body, err := ioutil.ReadAll(resp.Body)
					if err != nil {
						wg.Done()
						return err
					}
					file, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
					if err != nil {
						wg.Done()
						return err
					}
					defer file.Close()
					io.Copy(file, resp.Body)
					file.WriteAt(body, int64(start))
					wg.Done()
					return nil
				}(start, end, i)
			}
			wg.Wait()
			return nil
		}
	}
}
