package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/winterssy/sreq"
	"runtime"
	"sync"
)

// assetsindex_files
var indexassets_waitgroup sync.WaitGroup
var system_info = runtime.GOOS
var system_arch = runtime.GOARCH

func main() {
	//Default返回一个默认的路由引擎
	r := gin.Default()
	//req.SetTimeout(time.Second * 1800)
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
		Download_game(c.PostForm("MCVersion"), c.PostForm("VersionName"))
	})

	//启动MineCraft
	r.POST("/execute", func(c *gin.Context) {
		java_path := "'D:\\JavaJDK\\bin\\java.exe' "
		version := c.PostForm("MCVersion")
		version_name := c.PostForm("VersionName")
		user_name := c.PostForm("UserName")
		Execute_game(version, version_name, user_name, java_path)
	})

	/*
		运行代理
	*/
	r.Run(":30713")
}
