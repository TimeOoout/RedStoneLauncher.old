package main

import (
	"archive/zip"
	"bufio"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/winterssy/sreq"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// assetsindex_files
var indexassets_waitgroup sync.WaitGroup
var system_info = runtime.GOOS
var batchCh = make(chan struct{}, 10)

func main() {
	//Default返回一个默认的路由引擎
	r := gin.Default()

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
			go download_obj(hash)
			return true
		})
		///*依赖库文件下载*/
		//os.MkdirAll(".minecraft/libraries/", os.ModePerm)
		//os.MkdirAll(".minecraft/versions/"+c.PostForm("VersionName")+"/natives", os.ModePerm)
		//lib_num, _ := strconv.Atoi(gjson.Get(version_file_content, "libraries.#").String())
		//for i := 0; i < lib_num; i++ {
		//	go download_library(gjson.Get(version_file_content, "libraries."+strconv.Itoa(i)).String(), c.PostForm("VersionName"))
		//}
		//下载主文件
		//go download_assets(c.PostForm("VersionName"), gjson.Get(version_file_content, "downloads.client.url").String(), c.PostForm("Version"))

		indexassets_waitgroup.Wait()
		os.Remove(".minecraft/versions/" + c.PostForm("VersionName") + "/natives/native.jar")
	})
	/*
		运行代理
	*/
	r.Run(":30713")
}

func download_assets(version_name string, url string, v string) {
	indexassets_waitgroup.Add(1)
	asset_content, _ := sreq.Get(url).Content()
	asset_file, _ := os.OpenFile(".minecraft/versions/"+version_name+"/"+v+".jar", os.O_CREATE|os.O_TRUNC, 0666)
	bufWriter := bufio.NewWriter(asset_file)
	bufWriter.Write(asset_content)
	bufWriter.Flush()
	defer asset_file.Close()

	indexassets_waitgroup.Done()
	runtime.Gosched()
}

func download_obj(h string) {
	indexassets_waitgroup.Add(1)
	os.MkdirAll(".minecraft/assets/objects/"+h[0:2], os.ModePerm)
	obj_file_content, _ := sreq.Get("http://resources.download.minecraft.net/" + h[0:2] + "/" + h).Content()
	obj_file, _ := os.OpenFile(".minecraft/assets/objects/"+h[0:2]+"/"+h, os.O_CREATE|os.O_TRUNC, 0666)
	bufWriter := bufio.NewWriter(obj_file)
	bufWriter.Write(obj_file_content)
	bufWriter.Flush()
	defer obj_file.Close()

	indexassets_waitgroup.Done()
	runtime.Gosched()
}

func download_library(h string, v string) {
	indexassets_waitgroup.Add(1)
	path_ := gjson.Get(h, "downloads.artifact.path").String()
	path_list := strings.Split(path_, "/")
	lib_filename := path_list[len(path_list)-1]
	path_list = path_list[:len(path_list)-1]
	path := strings.Join(path_list, "/")

	os.MkdirAll(".minecraft/libraries/"+path, os.ModePerm)
	lib_url := gjson.Get(h, "downloads.artifact.url").String()
	lib_content, _ := sreq.Get(lib_url).Content()
	lib_file, _ := os.OpenFile(".minecraft/libraries/"+path+"/"+lib_filename, os.O_CREATE|os.O_TRUNC, 0666)
	bufWriter := bufio.NewWriter(lib_file)
	bufWriter.Write(lib_content)
	bufWriter.Flush()
	defer lib_file.Close()

	if gjson.Get(h, "downloads.classifiers").String() != "" {
		var native_url string
		if system_info == "windows" {
			native_url = gjson.Get(h, "downloads.classifiers.natives-windows.url").String()
		} else if system_info == "darwin" {
			native_url = gjson.Get(h, "downloads.classifiers.natives-osx.url").String()
		} else {
			native_url = gjson.Get(h, "downloads.classifiers.natives-linux.url").String()
		}
		native_content, _ := sreq.Get(native_url).Content()
		native_file, _ := os.OpenFile(".minecraft/versions/"+v+"/natives/native.jar", os.O_CREATE|os.O_TRUNC, 0666)
		bufWriter2 := bufio.NewWriter(native_file)
		bufWriter2.Write(native_content)
		bufWriter2.Flush()
		defer native_file.Close()
		go Unzip(".minecraft/versions/"+v+"/natives/native.jar", ".minecraft/versions/"+v+"/natives/")
	}
	indexassets_waitgroup.Done()
	runtime.Gosched()
}

func Unzip(zipFile string, destDir string) error {
	indexassets_waitgroup.Add(1)
	zipReader, err := zip.OpenReader(zipFile)
	defer indexassets_waitgroup.Done()
	defer runtime.Gosched()
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
